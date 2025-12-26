--[[
    VinzHub Inventory Sync Module v2.0
    Syncs player inventory data to VinzHub REST API
    
    Features:
    - Full inventory sync (fish, rods, baits)
    - Fish images via Roblox CDN
    - Player stats (coins, level, equipped items)
    
    API Endpoint:
    - POST /api/v1/inventory/{roblox_user_id}/sync
    
    Usage:
        local InventorySync = loadstring(...)()
        InventorySync.Init()
]]

--------------------------------------------------------------------------------
-- MODULE
--------------------------------------------------------------------------------

local InventorySync = {}

--------------------------------------------------------------------------------
-- CONFIGURATION
--------------------------------------------------------------------------------

local Config = {
    APIBase = "https://api-v2.vinzhub.com/api/v1",
    Token = nil,            -- Session token (set via SetToken())
    SyncInterval = 300,     -- Seconds between auto-sync (5 min to reduce DB load)
    Debug = false,          -- Enable debug logging
    FetchIcons = true,      -- Fetch fish icons from Roblox API
    
    -- Tier filter: Only sync fish at or above this tier
    -- 1=Common, 2=Uncommon, 3=Rare, 4=Epic, 5=Legendary, 6=Mythical, 7=Secret, 8=Exotic
    MinTierToSync = 5,      -- 5 = Legendary and above only
}


-- Tier names for reference
local TierOrder = {
    ["Common"] = 1,
    ["Uncommon"] = 2,
    ["Rare"] = 3,
    ["Epic"] = 4,
    ["Legendary"] = 5,
    ["Mythical"] = 6,
    ["Secret"] = 7,
    ["Exotic"] = 8,
}

local function GetTierNumber(tierName)
    if type(tierName) == "number" then return tierName end
    return TierOrder[tierName] or 0
end


--------------------------------------------------------------------------------
-- SERVICES
--------------------------------------------------------------------------------

local HttpService = game:GetService("HttpService")
local Players = game:GetService("Players")
local ReplicatedStorage = game:GetService("ReplicatedStorage")

local Player = Players.LocalPlayer
local RobloxUserId = tostring(Player.UserId)

--------------------------------------------------------------------------------
-- STATE
--------------------------------------------------------------------------------

-- Singleton protection: If script is executed multiple times, only first runs
local SINGLETON_KEY = "_VINZHUB_INVENTORY_SYNC_INSTANCE"
if _G[SINGLETON_KEY] then
    -- Stop old instance's sync loop
    if Config.Debug then
        warn("[InventorySync] Re-executing, stopping old instance...")
    end
    _G[SINGLETON_KEY].Stop()  -- Stop old sync loop
    _G[SINGLETON_KEY] = nil   -- Clear old instance
end

local SyncRunning = false
local AutoSyncStarted = false  -- Prevent multiple auto-sync loops
local SyncStopped = false      -- Circuit breaker: stop sync on repeated failures
local ConsecutiveErrors = 0    -- Track consecutive API errors
local MaxConsecutiveErrors = 3 -- Stop sync after this many failures
local LastSync = 0
local IconCache = {}  -- Cache for icon URLs to avoid repeated API calls


--------------------------------------------------------------------------------
-- GAME MODULES (lazy loaded)
--------------------------------------------------------------------------------

local ItemUtility = nil
local Replion = nil
local DataService = nil
local RollData = nil

local function InitGameModules()
    if DataService then return true end
    
    local success = pcall(function()
        ItemUtility = require(ReplicatedStorage.Shared.ItemUtility)
        Replion = require(ReplicatedStorage.Packages.Replion)
        DataService = Replion.Client:WaitReplion("Data")
    end)
    
    -- Load RollData separately to catch any errors
    local rollSuccess, rollError = pcall(function()
        RollData = require(ReplicatedStorage.Shared.RollData)
    end)
    
    if Config.Debug then
        print("[InventorySync] Game modules:", success and "loaded" or "failed")
    end
    
    return success
end

--------------------------------------------------------------------------------
-- HTTP REQUEST HELPER
--------------------------------------------------------------------------------

local function Request(method, endpoint, body)
    -- Validate token is set
    if not Config.Token then
        if Config.Debug then warn("[InventorySync] No token set! Call SetToken() first.") end
        return false, "Token not set"
    end
    
    local url = Config.APIBase .. endpoint
    
    if Config.Debug then 
        print("[InventorySync]", method, endpoint) 
    end
    
    local headers = {
        ["Content-Type"] = "application/json",
        ["X-Token"] = Config.Token  -- Use session token instead of API key
    }

    
    local requestData = {
        Url = url,
        Method = method,
        Headers = headers,
        Body = body
    }
    
    -- Find available HTTP request function
    local httpFunc = (syn and syn.request) 
        or request 
        or http_request 
        or (http and http.request)
    
    if not httpFunc then
        if Config.Debug then warn("[InventorySync] No HTTP function available") end
        return false, "No HTTP function"
    end
    
    local success, response = pcall(httpFunc, requestData)
    
    if not success then
        if Config.Debug then warn("[InventorySync] Request error:", response) end
        return false, response
    end
    
    if Config.Debug then 
        print("[InventorySync] Response:", response.StatusCode) 
    end
    
    return response.StatusCode == 200, response
end

--------------------------------------------------------------------------------
-- ICON UTILITIES
--------------------------------------------------------------------------------

local function GetThumbnailURL(assetString)
    if not assetString or not Config.FetchIcons then return nil end
    
    local assetId = tostring(assetString):match("rbxassetid://(%d+)")
    if not assetId then return nil end
    
    -- Check cache first
    if IconCache[assetId] then
        return IconCache[assetId]
    end
    
    local api = string.format(
        "https://thumbnails.roblox.com/v1/assets?assetIds=%s&returnPolicy=PlaceHolder&size=420x420&format=Png",
        assetId
    )
    
    local success, response = pcall(function()
        local raw = game:HttpGet(api)
        return HttpService:JSONDecode(raw)
    end)
    
    if success and response and response.data and response.data[1] then
        local imageUrl = response.data[1].imageUrl
        IconCache[assetId] = imageUrl  -- Cache for future use
        return imageUrl
    end
    
    return nil
end

--------------------------------------------------------------------------------
-- DATA EXTRACTION
--------------------------------------------------------------------------------

local function GetFishDetails(fishId)
    if not ItemUtility then return nil end
    
    local success, itemData = pcall(function()
        return ItemUtility.GetItemDataFromItemType("Items", fishId)
    end)
    
    if not success or not itemData or not itemData.Data then
        return nil
    end
    
    return {
        name = itemData.Data.Name,
        tier = itemData.Data.Tier,
        icon = GetThumbnailURL(itemData.Data.Icon),
        type = itemData.Data.Type,
        price = itemData.SellPrice
    }
end

local function GetBaitDetails(baitId, baitType)
    if not ItemUtility then return nil end
    
    -- Get basic bait info from ItemUtility
    local success, baitData = pcall(function()
        return ItemUtility.GetItemDataFromItemType("Baits", baitId)
    end)
    
    if not success or not baitData then
        -- Try alternative method
        success, baitData = pcall(function()
            return ItemUtility.GetBaitData(ItemUtility, baitId)
        end)
    end
    
    local name = "Unknown Bait"
    local tier = 1
    local icon = nil
    local baitTypeName = nil
    
    if baitData then
        local data = baitData.Data or baitData
        name = data.Name or name
        tier = data.Tier or tier
        icon = GetThumbnailURL(data.Icon)
        -- The bait type for RollData lookup should be the name (e.g., "Golden", "Supercharged")
        baitTypeName = data.Name or data.Type or baitType
    end
    
    -- Get modifiers from RollData based on bait type NAME (not the passed baitType)
    local modifiers = {}
    local lookupType = baitTypeName or baitType
    
    -- Strip " Bait" suffix for RollData lookup (e.g., "Corrupt Bait" → "Corrupt")
    if lookupType then
        lookupType = string.gsub(lookupType, " Bait$", "")
        lookupType = string.gsub(lookupType, "Bait$", "") -- Also try without space
    end
    
    if RollData and RollData.Modifiers and lookupType then
        local rollModifiers = RollData.Modifiers[lookupType]
        
        if rollModifiers then
            -- Extract modifier values from RollData
            for key, value in pairs(rollModifiers) do
                -- Handle nested tables (e.g., if value is a table with more data)
                local actualValue = value
                if typeof(value) == "table" then
                    -- Try to extract the numeric value
                    actualValue = value.Value or value.Multiplier or value[1]
                end
                
                if typeof(actualValue) == "number" then
                    local lowerKey = string.lower(key)
                    if lowerKey == "luck" or string.find(lowerKey, "luck") then
                        -- Convert luck multiplier to percentage if needed
                        if actualValue > 10 then
                            modifiers.luck = math.floor(actualValue)
                        else
                            modifiers.luck = math.floor(actualValue * 100)
                        end
                    elseif lowerKey == "mutation" or string.find(lowerKey, "mutation") then
                        modifiers.mutation_chance = actualValue
                    elseif lowerKey == "shiny" or string.find(lowerKey, "shiny") then
                        modifiers.shiny_chance = actualValue
                    elseif lowerKey == "fairy" or string.find(lowerKey, "fairy") then
                        modifiers.fairy_dust = actualValue
                    elseif lowerKey == "gold" or string.find(lowerKey, "coin") then
                        modifiers.gold = actualValue
                    elseif lowerKey == "xp" or string.find(lowerKey, "exp") then
                        modifiers.bonus_xp = actualValue
                    else
                        -- Store any other modifiers with their original key
                        modifiers[lowerKey] = actualValue
                    end
                end
            end
        end
    end
    
    return {
        name = name,
        tier = tier,
        icon = icon,
        bait_type = lookupType,
        modifiers = modifiers
    }
end

local function GetRodDetails(rodId)
    if not ItemUtility then return nil end
    
    local success, rodData = pcall(function()
        return ItemUtility.GetItemDataFromItemType("Fishing Rods", rodId)
    end)
    
    if not success or not rodData or not rodData.Data then
        return nil
    end
    
    -- Extract roll data (luck, click power, max weight)
    local rollData = rodData.RollData or {}
    local baseLuck = rollData.BaseLuck or 1
    local clickPower = rodData.ClickPower or 0.05
    local maxWeight = rodData.MaxWeight or 5
    
    -- Calculate display values like in game
    local luckPercent = math.floor(baseLuck * 100)
    local speedPercent = rodData.VisualClickPowerPercent and 
        math.round(rodData.VisualClickPowerPercent * 100) or 
        math.round(((clickPower * 25) ^ 2.5))
    
    return {
        name = rodData.Data.Name,
        tier = rodData.Data.Tier,
        icon = GetThumbnailURL(rodData.Data.Icon),
        modifiers = rodData.Modifiers,
        is_skin = rodData.IsSkin or false,
        equip_as_skin = rodData.EquipAsSkin or false,
        stats = {
            luck = luckPercent,
            speed = speedPercent,
            max_weight = maxWeight
        }
    }
end

--------------------------------------------------------------------------------
-- DATA COLLECTION
--------------------------------------------------------------------------------

local function CollectInventoryData()
    if not InitGameModules() then 
        return nil 
    end
    
    local data = {
        player = {
            user_id = RobloxUserId,
            username = Player.Name,
            display_name = Player.DisplayName
        },
        fish = {},
        rods = {},
        baits = {},
        potions = {},
        stones = {},      -- Enchant Stones
        gears = {},       -- Gears (from Items)
        trophies = {},    -- Trophies (from Items)
        boats = {},       -- Boats
        totems = {},      -- Totems
        lanterns = {},    -- Lanterns
        stats = {},
        synced_at = os.time()
    }
    
    local success = pcall(function()
        local inventory = DataService:GetExpect({"Inventory"})
        if not inventory then return end
        
        -- Process Items (Fish, Enchant Stones, Gears, Trophies are all in Items)
        if inventory.Items then
            for _, item in pairs(inventory.Items) do
                local details = GetFishDetails(item.Id)
                if details then
                    local itemType = details.type
                    local itemData = {
                        uuid = item.UUID,
                        item_id = item.Id,
                        name = details.name,
                        tier = details.tier,
                        icon = details.icon,
                        price = (details.price and details.price > 0) and details.price or nil,
                        favorited = item.Favorited or false,
                        is_shiny = item.Shiny or false,
                        metadata = item.Metadata,
                        variant_id = item.Metadata and item.Metadata.VariantId,
                        quantity = item.Quantity or 1
                    }
                    
                    if itemType == "Fish" then
                        -- Filter: Only sync fish at or above MinTierToSync
                        local tierNum = GetTierNumber(details.tier)
                        if tierNum >= Config.MinTierToSync then
                            itemData.fish_id = item.Id
                            table.insert(data.fish, itemData)
                        end
                    elseif itemType == "Enchant Stones" or itemType == "Enchant Stone" or (details.name and details.name:find("Enchant")) then
                        itemData.stone_id = item.Id
                        table.insert(data.stones, itemData)
                    elseif itemType == "Gears" or itemType == "Gear" then
                        itemData.gear_id = item.Id
                        table.insert(data.gears, itemData)
                    elseif itemType == "Trophies" or itemType == "Trophy" then
                        itemData.trophy_id = item.Id
                        table.insert(data.trophies, itemData)
                    end
                end
            end
        end
        
        -- Process Rods
        if inventory["Fishing Rods"] then
            for _, rod in pairs(inventory["Fishing Rods"]) do
                local details = GetRodDetails(rod.Id)
                if details then
                    table.insert(data.rods, {
                        uuid = rod.UUID,
                        rod_id = rod.Id,
                        name = details.name,
                        tier = details.tier,
                        icon = details.icon,
                        equipped = rod.Equipped or false,
                        is_skin = details.is_skin,
                        stats = details.stats,
                        modifiers = details.modifiers,
                        metadata = rod.Metadata  -- Contains EnchantId, EnchantId2, VariantId
                    })
                end
            end
        end
        
        -- Process Baits
        if inventory.Baits then
            for _, bait in pairs(inventory.Baits) do
                -- Extract bait type (Golden, Supercharged, Galactic, Corrupt, etc.)
                local baitType = bait.Type or bait.BaitType or bait.Name
                local details = GetBaitDetails(bait.Id, baitType)
                
                table.insert(data.baits, {
                    uuid = bait.UUID,
                    bait_id = bait.Id,
                    name = details and details.name or baitType or ("Bait " .. bait.Id),
                    tier = details and details.tier,
                    icon = details and details.icon,
                    bait_type = details and details.bait_type,
                    quantity = bait.Quantity,
                    equipped = bait.Equipped or false,
                    modifiers = details and details.modifiers
                })
            end
        end
        
        -- Process Potions
        if inventory.Potions then
            for _, potion in pairs(inventory.Potions) do
                local potionData = ItemUtility.GetItemDataFromItemType("Potions", potion.Id)
                table.insert(data.potions, {
                    uuid = potion.UUID,
                    potion_id = potion.Id,
                    name = potionData and potionData.Data and potionData.Data.Name or ("Potion " .. potion.Id),
                    tier = potionData and potionData.Data and potionData.Data.Tier,
                    icon = potionData and potionData.Data and GetThumbnailURL(potionData.Data.Icon),
                    quantity = potion.Quantity or 1
                })
            end
        end
        
        -- Process Totems
        if inventory.Totems then
            for _, totem in pairs(inventory.Totems) do
                local totemData = ItemUtility.GetItemDataFromItemType("Totems", totem.Id)
                table.insert(data.totems, {
                    uuid = totem.UUID,
                    totem_id = totem.Id,
                    name = totemData and totemData.Data and totemData.Data.Name or ("Totem " .. totem.Id),
                    tier = totemData and totemData.Data and totemData.Data.Tier,
                    icon = totemData and totemData.Data and GetThumbnailURL(totemData.Data.Icon),
                    quantity = totem.Quantity or 1,
                    metadata = totem.Metadata
                })
            end
        end
        
        -- Process Boats
        if inventory.Boats then
            for _, boat in pairs(inventory.Boats) do
                local boatData = ItemUtility.GetItemDataFromItemType("Boats", boat.Id)
                table.insert(data.boats, {
                    uuid = boat.UUID,
                    boat_id = boat.Id,
                    name = boatData and boatData.Data and boatData.Data.Name or ("Boat " .. boat.Id),
                    tier = boatData and boatData.Data and boatData.Data.Tier,
                    icon = boatData and boatData.Data and GetThumbnailURL(boatData.Data.Icon),
                    equipped = boat.Equipped or false
                })
            end
        end
        
        -- Process Lanterns
        if inventory.Lanterns then
            for _, lantern in pairs(inventory.Lanterns) do
                local lanternData = ItemUtility.GetItemDataFromItemType("Lanterns", lantern.Id)
                table.insert(data.lanterns, {
                    uuid = lantern.UUID,
                    lantern_id = lantern.Id,
                    name = lanternData and lanternData.Data and lanternData.Data.Name or ("Lantern " .. lantern.Id),
                    tier = lanternData and lanternData.Data and lanternData.Data.Tier,
                    icon = lanternData and lanternData.Data and GetThumbnailURL(lanternData.Data.Icon),
                    equipped = lantern.Equipped or false
                })
            end
        end
        
        -- Player Stats
        data.stats = {
            coins = DataService:Get("Coins") or 0,
            level = DataService:Get("Level") or 0,
            equipped_items = DataService:GetExpect("EquippedItems") or {},
            equipped_bait_id = DataService:GetExpect("EquippedBaitId"),
            auto_sell_threshold = DataService:Get("AutoSellThreshold")
        }
    end)
    
    if Config.Debug then
        print(string.format(
            "[InventorySync] Collected: %d fish, %d rods, %d baits, %d potions, %d stones, %d gears, %d trophies, %d boats, %d totems, %d lanterns",
            #data.fish, #data.rods, #data.baits, #data.potions, #data.stones,
            #data.gears, #data.trophies, #data.boats, #data.totems, #data.lanterns
        ))
    end
    
    return data
end

--------------------------------------------------------------------------------
-- PUBLIC API
--------------------------------------------------------------------------------

function InventorySync.Sync()
    -- Circuit breaker check
    if SyncStopped then
        if Config.Debug then
            warn("[InventorySync] Sync stopped due to repeated failures")
        end
        return false, "Sync stopped"
    end
    
    -- Token check
    if not Config.Token then
        if Config.Debug then
            warn("[InventorySync] No token, skipping sync")
        end
        return false, "No token"
    end
    
    if SyncRunning then 
        return false, "Sync already running" 
    end
    
    SyncRunning = true
    
    local data = CollectInventoryData()
    if not data then
        SyncRunning = false
        return false, "Failed to collect data"
    end
    
    local jsonBody = HttpService:JSONEncode(data)
    local endpoint = "/inventory/" .. RobloxUserId .. "/sync"
    local success, response = Request("POST", endpoint, jsonBody)
    
    SyncRunning = false
    LastSync = os.time()
    
    -- Circuit breaker: track errors
    if success then
        ConsecutiveErrors = 0  -- Reset on success
    else
        ConsecutiveErrors = ConsecutiveErrors + 1
        if ConsecutiveErrors >= MaxConsecutiveErrors then
            SyncStopped = true
            warn(string.format("[InventorySync] Sync stopped after %d consecutive errors", ConsecutiveErrors))
        end
    end
    
    if Config.Debug then
        print("[InventorySync] Sync:", success and "complete" or "failed")
    end
    
    return success, response
end

function InventorySync.StartAutoSync()
    -- Prevent multiple auto-sync loops
    if AutoSyncStarted then
        if Config.Debug then
            warn("[InventorySync] AutoSync already running, ignoring duplicate call")
        end
        return
    end
    AutoSyncStarted = true
    
    task.spawn(function()
        task.wait(5)  -- Initial delay
        InventorySync.Sync()
        
        while not SyncStopped do  -- Stop loop if circuit breaker triggered
            task.wait(Config.SyncInterval)
            if SyncStopped then break end
            InventorySync.Sync()
        end
        
        if SyncStopped then
            warn("[InventorySync] AutoSync loop terminated due to errors")
        end
    end)
    
    if Config.Debug then
        print("[InventorySync] AutoSync started (interval:", Config.SyncInterval, "s)")
    end
end

-- Reset circuit breaker (can be called to retry after fixing issues)
function InventorySync.ResetCircuitBreaker()
    SyncStopped = false
    ConsecutiveErrors = 0
    if Config.Debug then
        print("[InventorySync] Circuit breaker reset")
    end
end

-- Stop sync loop (called when re-executing script)
function InventorySync.Stop()
    SyncStopped = true
    AutoSyncStarted = false
    Config.Token = nil
    if Config.Debug then
        print("[InventorySync] Stopped")
    end
end

function InventorySync.GetLastSync()
    return LastSync
end

--------------------------------------------------------------------------------
-- CONFIGURATION API
--------------------------------------------------------------------------------

function InventorySync.SetToken(token)
    Config.Token = token
end

-- Login with License Key (Auto-fetches Token)
function InventorySync.Login(licenseKey, manualHwid)
    local hwid = manualHwid
    
    -- Try to get HWID from executor if not provided
    if not hwid and gethwid then
        hwid = gethwid()
    end
    
    -- Fallback/Validate HWID
    if not hwid then
        warn("[InventorySync] Warning: No HWID found. Using placeholder 'unknown-hwid'")
        hwid = "unknown-hwid"
    end

    local url = Config.APIBase .. "/auth/token"
    local body = HttpService:JSONEncode({
        key = licenseKey,
        hwid = hwid,
        roblox_id = RobloxUserId
    })
    
    if Config.Debug then
        print("[InventorySync] Logging in with Key:", licenseKey, "HWID:", hwid)
    end

    local requestData = {
        Url = url,
        Method = "POST",
        Headers = {
            ["Content-Type"] = "application/json"
        },
        Body = body
    }
    
    -- Find available HTTP request function
    local httpFunc = (syn and syn.request) 
        or request 
        or http_request 
        or (http and http.request)
        
    if not httpFunc then
        warn("[InventorySync] Login Failed: No HTTP function available")
        return false
    end
    
    local success, response = pcall(httpFunc, requestData)
    
    if success and response.StatusCode == 200 then
        local data = HttpService:JSONDecode(response.Body)
        -- API v2 returns {success: true, data: {token, expires_in}}
        local tokenData = data.data or data  -- Support both wrapped and unwrapped
        if tokenData and tokenData.token then
            Config.Token = tokenData.token
            if Config.Debug then
                print("[InventorySync] Login Successful! Token expires in:", tokenData.expires_in)
            end
            return true
        else
            warn("[InventorySync] Login Failed: Invalid response format")
        end
    else
        local body = response and response.Body or "Unknown Error"
        -- Check if body is HTML (Cloudflare error, etc)
        if type(body) == "string" and (body:match("^%s*<!doctype") or body:match("^%s*<html") or body:match("Cloudflare")) then
            warn(string.format("[InventorySync] Login Failed: Server Error %s (Cloudflare/Gateway Error)", tostring(response and response.StatusCode or "Unknown")))
        else
            warn("[InventorySync] Login Failed:", body)
        end
    end
    return false
end

-- Deprecated: Use SetLicenseKey(key) or SetToken(token)
function InventorySync.SetAPIKey(key)
    warn("[InventorySync] SetAPIKey is deprecated. Attempting auto-login...")
    InventorySync.Login(key)
end

-- Set license key and auto-login to get token from API v2
function InventorySync.SetLicenseKey(key)
    InventorySync.Login(key)
end

function InventorySync.SetAPIBase(url)
    Config.APIBase = url
end

function InventorySync.SetSyncInterval(seconds)
    Config.SyncInterval = seconds
end

function InventorySync.EnableDebug(enabled)
    Config.Debug = enabled
end

function InventorySync.EnableIcons(enabled)
    Config.FetchIcons = enabled
end

function InventorySync.SetMinTier(tier)
    Config.MinTierToSync = tier
end

--------------------------------------------------------------------------------
-- INITIALIZATION
--------------------------------------------------------------------------------

function InventorySync.Init()
    -- Validate token is set
    if not Config.Token then
        warn("[InventorySync] ERROR: Token not set! Call SetToken(token) before Init().")
        warn("[InventorySync] Get token from: POST /auth/token {key, hwid, roblox_id}")
        SyncStopped = true  -- Prevent any sync attempts
        return false
    end
    
    if Config.Debug then
        print("[InventorySync] v3.0 initialized for user:", RobloxUserId)
    end
    InventorySync.StartAutoSync()
    return true
end

-- Register singleton before returning
_G[SINGLETON_KEY] = InventorySync

return InventorySync

