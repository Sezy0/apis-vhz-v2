if _G.VinzHubUI and _G.VinzHubUI.Parent then
    _G.VinzHubUI:Destroy()
end
_G.VinzHubUI = nil

local HttpService = game:GetService("HttpService")
local RbxAnalyticsService = game:GetService("RbxAnalyticsService")
local Players = game:GetService("Players")
local TweenService = game:GetService("TweenService")
local UserInputService = game:GetService("UserInputService")
local StarterGui = game:GetService("StarterGui")

local Player = Players.LocalPlayer
local PlayerGui = Player:WaitForChild("PlayerGui")

for _, gui in pairs(PlayerGui:GetChildren()) do
    if gui.Name == "VinzHubKeySystem" then gui:Destroy() end
end

local Config = {
    APIBase         = "https://api.vinzhub.com",
    DiscordLink     = "https://discord.gg/vinzhub",
    GetKeyLink      = "https://vinzhub.com/getkey",
    ScriptName      = "VinzHub",
    Version         = "3.0",
    HeartbeatInterval = 60,
    Colors = {
        Background      = Color3.fromRGB(15, 15, 20),
        BackgroundAlt   = Color3.fromRGB(20, 20, 28),
        Glass           = Color3.fromRGB(30, 30, 40),
        GlassHover      = Color3.fromRGB(40, 40, 55),
        Accent          = Color3.fromRGB(99, 102, 241),
        AccentHover     = Color3.fromRGB(129, 140, 248),
        Text            = Color3.fromRGB(250, 250, 255),
        TextMuted       = Color3.fromRGB(140, 140, 160),
        TextDim         = Color3.fromRGB(90, 90, 110),
        Border          = Color3.fromRGB(50, 50, 65),
        Success         = Color3.fromRGB(34, 197, 94),
        Error           = Color3.fromRGB(239, 68, 68),
    }
}

-- ═══════════════════════════════════════════════════════════════
-- EXECUTION TRACKING SYSTEM
-- ═══════════════════════════════════════════════════════════════

local function GetExecutorInfo()
    local info = {
        name = "Unknown",
        version = "Unknown"
    }
    
    pcall(function()
        -- Detect common executors
        if identifyexecutor then
            local name, ver = identifyexecutor()
            info.name = name or "Unknown"
            info.version = ver or "Unknown"
        elseif getexecutorname then
            info.name = getexecutorname()
        elseif _G.ExecutorName then
            info.name = _G.ExecutorName
        end
        
        -- Fallback detection
        if info.name == "Unknown" then
            if syn then info.name = "Synapse X"
            elseif KRNL_LOADED then info.name = "KRNL"
            elseif fluxus then info.name = "Fluxus"
            elseif getgenv().Hydrogen then info.name = "Hydrogen"
            elseif Wave then info.name = "Wave"
            elseif Solara then info.name = "Solara"
            elseif Evon then info.name = "Evon"
            elseif Delta then info.name = "Delta"
            end
        end
    end)
    
    return info
end

local function GetPlatform()
    local platform = "Unknown"
    pcall(function()
        local userAgent = game:GetService("UserInputService"):GetPlatform()
        if userAgent then
            platform = tostring(userAgent):gsub("Enum.Platform.", "")
        end
    end)
    return platform
end

local function GenerateNonce()
    local chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    local nonce = ""
    for i = 1, 32 do
        local idx = math.random(1, #chars)
        nonce = nonce .. chars:sub(idx, idx)
    end
    return nonce
end

local function TrackExecution(scriptId, scriptName, status, errorMsg, key)
    task.spawn(function()
        local success, err = pcall(function()
            local hwid = RbxAnalyticsService:GetClientId()
            local executorInfo = GetExecutorInfo()
            local timestamp = os.time()
            local nonce = GenerateNonce()
            
            local trackingData = {
                -- Executor & User Identity
                executor_name = executorInfo.name,
                executor_version = executorInfo.version,
                user_id = Player.UserId,
                username = Player.Name,
                display_name = Player.DisplayName,
                hwid = hwid,
                
                -- Request Security
                timestamp = timestamp,
                nonce = nonce,
                key = key or "",
                
                -- Environment & Client Context
                platform = GetPlatform(),
                place_id = game.PlaceId,
                job_id = game.JobId,
                game_name = game:GetService("MarketplaceService"):GetProductInfo(game.PlaceId).Name or "Unknown",
                
                -- Script Execution Metadata
                script_id = scriptId,
                script_name = scriptName,
                status = status, -- "success" | "failed" | "blocked"
                error_message = errorMsg or "",
                
                -- Additional Context
                account_age = Player.AccountAge,
                membership = tostring(Player.MembershipType):gsub("Enum.MembershipType.", ""),
                locale = game:GetService("LocalizationService").RobloxLocaleId,
            }
            
            local jsonData = HttpService:JSONEncode(trackingData)
            local url = "https://script.vinzhub.com/api/track-execution"
            
            local response = nil
            
            if syn and syn.request then
                response = syn.request({
                    Url = url,
                    Method = "POST",
                    Headers = {["Content-Type"] = "application/json"},
                    Body = jsonData
                })
            elseif request then
                response = request({
                    Url = url,
                    Method = "POST",
                    Headers = {["Content-Type"] = "application/json"},
                    Body = jsonData
                })
            elseif http_request then
                response = http_request({
                    Url = url,
                    Method = "POST",
                    Headers = {["Content-Type"] = "application/json"},
                    Body = jsonData
                })
            elseif http and http.request then
                response = http.request({
                    Url = url,
                    Method = "POST",
                    Headers = {["Content-Type"] = "application/json"},
                    Body = jsonData
                })
            else
                -- Fallback (Silent)
                local params = "?data=" .. HttpService:UrlEncode(jsonData)
                game:HttpGet(url .. params)
                return
            end
        end)
    end)
end

-- ═══════════════════════════════════════════════════════════════

local GameScripts = {
    [121864768012064] = {
        Name    = "Fish It",
        URL     = "https://script.vinzhub.com/execute/private?id=13&key=f60a77419d5c5027f33e77b113f0ca2e03e3e2be7f155c24299830586d3796a6",
        LiteURL = "https://script.vinzhub.com/execute/private?id=16&key=d1ee22b104f69ee7960973c163dd870a7c18eb28f612552acc5d3863a908669d"
    },
    [129009554587176] = {
        Name = "The Forge (Map 1)",
        URL  = "https://script.vinzhub.com/execute/private?id=17&key=8fa7f1d4436377e010bafea399cfaad35ec5ee644c4fbe3beb2b567c0adad4b0"
    },
    [76558904092080] = {
        Name = "The Forge (Map 2)",
        URL  = "https://script.vinzhub.com/execute/private?id=17&key=8fa7f1d4436377e010bafea399cfaad35ec5ee644c4fbe3beb2b567c0adad4b0"
    },
    [131884594917121] = {
        Name = "The Forge (Map 3)",
        URL  = "https://script.vinzhub.com/execute/private?id=17&key=8fa7f1d4436377e010bafea399cfaad35ec5ee644c4fbe3beb2b567c0adad4b0"
    },
    [123224294054165] = {
        Name = "Mount Atin",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/mt-atin"
    },
    [102234703920418] = {
        Name = "Mount Daun",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/mt-daun"
    },
    [128473079243102] = {
        Name = "Mount Arunika",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/mt-arunika"
    },
    [2693023319] = {
        Name = "Expedition Antartica",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/antartica"
    },
    [106525193781380] = {
        Name = "Mount Sibuatan",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/mt-sibuatan"
    },
    [93978595733734] = {
        Name = "Violance District",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/VD"
    },
    [127742093697776] = {
        Name = "Plant VS Brainrot",
        URL  = "https://raw.githubusercontent.com/Vinzyy13/VinzHub/refs/heads/main/PVB"
    },
}

local Session = {
    Key              = nil,
    MaxAccounts      = 0,
    AccountsUsed     = 0,
    KeyStatus        = nil,
    OnlineUsers      = 0,
    LastHeartbeat    = 0,
    HeartbeatRunning = false
}

local function Tween(obj, props, duration, style, direction)
    local info = TweenInfo.new(duration or 0.25, style or Enum.EasingStyle.Quint, direction or Enum.EasingDirection.Out)
    return TweenService:Create(obj, info, props)
end

local function CreateElement(class, props)
    local element = Instance.new(class)
    for prop, value in pairs(props) do
        element[prop] = value
    end
    return element
end

-- ═══════════════════════════════════════════════════════════════
-- VALIDATE KEY v3 - Full API Response Handling
-- Matches vinzhub-resapi/ValidateKeyController.php
-- ═══════════════════════════════════════════════════════════════
local function ValidateKey(key)
    key = key:gsub("%s+", "")
    if key == "" then 
        return false, "Key cannot be empty", nil 
    end

    local hwid = RbxAnalyticsService:GetClientId()
    local url = Config.APIBase .. "/validate-key?key=" .. key .. "&hwid=" .. hwid .. "&user_id=" .. Player.UserId .. "&username=" .. Player.Name
    
    local success, response = pcall(function() return game:HttpGet(url) end)
    
    if not success then 
        return false, "Connection failed", nil 
    end
    
    if response:sub(1, 15):lower():find("<!doctype") then 
        return false, "Security block. Try again.", nil 
    end

    -- Clean BOM and whitespace
    response = response:gsub("^\239\187\191", ""):gsub("^%s+", ""):gsub("%s+$", "")
    
    local ok, data = pcall(function() return HttpService:JSONDecode(response) end)
    
    if not ok then 
        return false, "Invalid response", nil 
    end
    
    -- ═══════════════════════════════════════════════════════════════
    -- API Response Cases (from ValidateKeyController.php):
    -- 1. Rate limit: {valid: false, error: "Rate limit exceeded", message: "..."}
    -- 2. Missing param: {valid: false, error: "Missing parameter", message: "..."}
    -- 3. Invalid format: {valid: false, error: "Invalid format", message: "..."}
    -- 4. Key not found: {valid: false, error: "Invalid key", message: "Key not found"}
    -- 5. Key inactive: {valid: false, error: "Key inactive", status: "...", message: "..."}
    -- 6. Account limit: {valid: false, error: "Account limit reached", message: "...", accounts_used: N, max_accounts: N}
    -- 7. Success: {valid: true, status: "active", max_accounts: N, accounts_used: N}
    -- ═══════════════════════════════════════════════════════════════
    
    if data.valid == true then
        -- Success response
        local sessionData = {
            status = data.status or "active",
            max_accounts = data.max_accounts or 1,
            accounts_used = data.accounts_used or 1
        }
        local successMsg = "Key validated! (" .. sessionData.accounts_used .. "/" .. sessionData.max_accounts .. " slots)"
        return true, successMsg, sessionData
    else
        -- Error response - handle all error types
        local errorType = data.error or "Unknown"
        local errorMsg = data.message or "Invalid key"
        
        -- Special handling for specific error types
        if errorType == "Rate limit exceeded" then
            errorMsg = "Too many requests. Wait a moment."
        elseif errorType == "Account limit reached" then
            local used = data.accounts_used or 0
            local max = data.max_accounts or 0
            errorMsg = "Slot penuh (" .. used .. "/" .. max .. ")"
        elseif errorType == "Key inactive" then
            local status = data.status or "inactive"
            errorMsg = "Key " .. status
        elseif errorType == "Invalid format" then
            errorMsg = "Format key salah"
        elseif errorType == "Invalid key" then
            errorMsg = "Key tidak ditemukan"
        end
        
        return false, errorMsg, nil
    end
end

local function GetGameData()
    return GameScripts[game.PlaceId]
end

local function HasLiteMode()
    local gameData = GetGameData()
    return gameData and gameData.LiteURL ~= nil
end

local function LoadGameScript(useLite, key)
    local gameData = GetGameData()
    
    if not gameData then 
        TrackExecution("unknown", "Unknown Game", "blocked", "Game not supported", key)
        return false, "Game not supported" 
    end
    
    _G.VinzHub_LoaderToken = "VH_" .. os.time() .. "_" .. math.random(100000, 999999)
    
    local targetURL = gameData.URL
    local modeName = "Normal"
    local scriptId = tostring(game.PlaceId)
    
    -- Logic Lite Mode
    if useLite and gameData.LiteURL then
        targetURL = gameData.LiteURL
        modeName = "Lite"
    end
    
    local success, result = pcall(function() return game:HttpGet(targetURL) end)
    
    if not success then 
        TrackExecution(scriptId, gameData.Name, "failed", "Network error", key)
        return false, "Network error" 
    end
    if result:sub(1, 15):lower():find("<!doctype") then 
        TrackExecution(scriptId, gameData.Name, "blocked", "Security block", key)
        return false, "Blocked by security" 
    end
    
    -- Compile script first
    local compiled, compileErr = loadstring(result)
    
    if not compiled then
        -- Script failed to compile (syntax error)
        local errMsg = tostring(compileErr):sub(1, 50)
        TrackExecution(scriptId, gameData.Name, "failed", "Compile: " .. errMsg, key)
        return false, "Compile error: " .. errMsg
    end
    
    -- Execute compiled script in spawn so runtime errors don't block loader
    -- This is because many scripts run continuously and may error later
    local initialSuccess = true
    local initialError = nil
    
    task.spawn(function()
        local execOk, execErr = pcall(compiled)
        if not execOk then
            -- Log runtime error but don't show notification (script may have partially worked)
            warn("[VinzHub] Script runtime error: " .. tostring(execErr))
            TrackExecution(scriptId, gameData.Name, "runtime_error", tostring(execErr), key)
        end
    end)
    
    -- Give script a moment to initialize and catch immediate errors
    task.wait(0.1)
    
    -- ═══════════════════════════════════════════════════════════════
    -- FISH IT INVENTORY SYNC - Auto-load for Fish It game
    -- ═══════════════════════════════════════════════════════════════
    local FISHIT_PLACE_ID = 121864768012064
    
    if game.PlaceId == FISHIT_PLACE_ID then
        task.spawn(function()
            pcall(function()
                -- Wait for game to fully load
                task.wait(3)
                
                -- Load InventorySync module
                local InventorySyncURL = "https://script.vinzhub.com/execute/private?id=21&key=aa5b5ad360931e175a506cc82e681d9e673bcf5eebdde47ee232edc78883a060"
                local syncModule = loadstring(game:HttpGet(InventorySyncURL))
                
                if syncModule then
                    local InventorySync = syncModule()
                    
                    if InventorySync and InventorySync.Init then
                        -- Use the validated key from loader (now in _G.script_key)
                        -- InventorySync will login and get token from API v2
                        if _G.script_key then
                            InventorySync.SetLicenseKey(_G.script_key)  -- Will auto-login
                        end
                        InventorySync.SetSyncInterval(60) -- Sync every 60 seconds
                        InventorySync.EnableDebug(false)
                        InventorySync.Init()
                        
                        -- Store globally for access from main script
                        _G.VinzHub_InventorySync = InventorySync
                        
                        print("[VinzHub] InventorySync loaded for Fish It (60s interval)")
                    end
                end
            end)
        end)
    end
    
    -- Track successful load (script started)
    TrackExecution(scriptId, gameData.Name .. " (" .. modeName .. ")", "success", nil, key)
    return true, gameData.Name .. " (" .. modeName .. ") loaded!"
end

-- ═══════════════════════════════════════════════════════════════
-- HEARTBEAT v3 - Full API Response Handling
-- Matches vinzhub-resapi/HeartbeatController.php
-- Response: {success: true, message: "Heartbeat received", online_users: N, timestamp: N}
-- ═══════════════════════════════════════════════════════════════
local function StartHeartbeat(key)
    if Session.HeartbeatRunning then return end
    Session.HeartbeatRunning = true
    Session.Key = key
    
    task.spawn(function()
        while Session.HeartbeatRunning do
            task.wait(Config.HeartbeatInterval)
            pcall(function()
                local hwid = RbxAnalyticsService:GetClientId()
                local url = Config.APIBase .. "/heartbeat?key=" .. key .. "&hwid=" .. hwid .. "&user_id=" .. Player.UserId .. "&username=" .. Player.Name
                
                local response = game:HttpGet(url)
                
                -- Parse response untuk mendapatkan online_users (optional)
                local ok, data = pcall(function() 
                    return HttpService:JSONDecode(response) 
                end)
                
                if ok and data then
                    if data.success then
                        -- Store online users count for potential UI display
                        Session.OnlineUsers = data.online_users or 0
                        Session.LastHeartbeat = data.timestamp or os.time()
                    elseif data.error == "Invalid key" then
                        -- Key became invalid, stop heartbeat
                        Session.HeartbeatRunning = false
                    end
                end
            end)
        end
    end)
end

local function StopHeartbeat()
    Session.HeartbeatRunning = false
end

local function CreateUI()
    if PlayerGui:FindFirstChild("VinzHubKeySystem") then
        PlayerGui:FindFirstChild("VinzHubKeySystem"):Destroy()
    end
    
    local C = Config.Colors
    
    local ScreenGui = CreateElement("ScreenGui", {
        Name = "VinzHubKeySystem",
        ResetOnSpawn = false,
        ZIndexBehavior = Enum.ZIndexBehavior.Sibling,
        Parent = PlayerGui
    })
    _G.VinzHubUI = ScreenGui
    
    local MainFrame = CreateElement("Frame", {
        Name = "MainFrame",
        Size = UDim2.new(0, 340, 0, 0),
        Position = UDim2.new(0.5, 0, 0.5, 0),
        AnchorPoint = Vector2.new(0.5, 0.5),
        BackgroundColor3 = C.Background,
        BackgroundTransparency = 0.05,
        BorderSizePixel = 0,
        ClipsDescendants = true,
        Parent = ScreenGui
    })
    
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 12), Parent = MainFrame })
    CreateElement("UIStroke", { Color = C.Border, Thickness = 1, Transparency = 0.5, Parent = MainFrame })
    
    local GradientOverlay = CreateElement("Frame", {
        Name = "GradientOverlay",
        Size = UDim2.new(1, 0, 0, 50),
        BackgroundColor3 = Color3.fromRGB(255, 255, 255),
        BackgroundTransparency = 0.97,
        BorderSizePixel = 0,
        Parent = MainFrame
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 12), Parent = GradientOverlay })
    
    local Header = CreateElement("Frame", {
        Name = "Header",
        Size = UDim2.new(1, 0, 0, 48),
        BackgroundTransparency = 1,
        Parent = MainFrame
    })
    
    local Logo = CreateElement("ImageLabel", {
        Name = "Logo",
        Size = UDim2.new(0, 20, 0, 20),
        Position = UDim2.new(0, 16, 0.5, 0),
        AnchorPoint = Vector2.new(0, 0.5),
        BackgroundTransparency = 1,
        Image = "rbxassetid://93128969335561",
        Parent = Header
    })
    
    local Title = CreateElement("TextLabel", {
        Name = "Title",
        Size = UDim2.new(1, -100, 1, 0),
        Position = UDim2.new(0, 42, 0, 0),
        BackgroundTransparency = 1,
        Text = Config.ScriptName,
        TextColor3 = C.Text,
        TextSize = 14,
        Font = Enum.Font.GothamBold,
        TextXAlignment = Enum.TextXAlignment.Left,
        Parent = Header
    })
    
    local VersionBadge = CreateElement("TextLabel", {
        Name = "Version",
        Size = UDim2.new(0, 32, 0, 16),
        Position = UDim2.new(0, 100, 0.5, 0),
        AnchorPoint = Vector2.new(0, 0.5),
        BackgroundColor3 = C.Glass,
        BackgroundTransparency = 0.3,
        Text = "v" .. Config.Version,
        TextColor3 = C.TextMuted,
        TextSize = 10,
        Font = Enum.Font.GothamMedium,
        Parent = Header
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 4), Parent = VersionBadge })
    
    local CloseBtn = CreateElement("TextButton", {
        Name = "CloseBtn",
        Size = UDim2.new(0, 28, 0, 28),
        Position = UDim2.new(1, -40, 0.5, 0),
        AnchorPoint = Vector2.new(0, 0.5),
        BackgroundColor3 = C.Glass,
        BackgroundTransparency = 0.5,
        Text = "",
        AutoButtonColor = false,
        Parent = Header
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 6), Parent = CloseBtn })
    
    local CloseIcon = CreateElement("ImageLabel", {
        Size = UDim2.new(0, 12, 0, 12),
        Position = UDim2.new(0.5, 0, 0.5, 0),
        AnchorPoint = Vector2.new(0.5, 0.5),
        BackgroundTransparency = 1,
        Image = "rbxassetid://10747384394",
        ImageColor3 = C.TextMuted,
        Parent = CloseBtn
    })
    
    CreateElement("Frame", {
        Name = "Divider",
        Size = UDim2.new(1, -32, 0, 1),
        Position = UDim2.new(0, 16, 0, 48),
        BackgroundColor3 = C.Border,
        BackgroundTransparency = 0.5,
        BorderSizePixel = 0,
        Parent = MainFrame
    })
    
    local Content = CreateElement("Frame", {
        Name = "Content",
        Size = UDim2.new(1, -32, 1, -60),
        Position = UDim2.new(0, 16, 0, 56),
        BackgroundTransparency = 1,
        Parent = MainFrame
    })
    
    local InputRow = CreateElement("Frame", {
        Name = "InputRow",
        Size = UDim2.new(1, 0, 0, 38),
        BackgroundTransparency = 1,
        Parent = Content
    })
    
    local InputFrame = CreateElement("Frame", {
        Name = "InputFrame",
        Size = UDim2.new(1, -80, 1, 0),
        BackgroundColor3 = C.Glass,
        BackgroundTransparency = 0.4,
        BorderSizePixel = 0,
        ClipsDescendants = true,
        Parent = InputRow
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 8), Parent = InputFrame })
    
    local InputStroke = CreateElement("UIStroke", {
        Color = C.Border,
        Thickness = 1,
        Transparency = 0.6,
        Parent = InputFrame
    })
    
    local KeyInput = CreateElement("TextBox", {
        Name = "KeyInput",
        Size = UDim2.new(1, -16, 1, 0),
        Position = UDim2.new(0, 8, 0, 0),
        BackgroundTransparency = 1,
        Text = "",
        PlaceholderText = "Enter key...",
        TextColor3 = C.Text,
        PlaceholderColor3 = C.TextDim,
        Font = Enum.Font.Gotham,
        TextSize = 12,
        TextXAlignment = Enum.TextXAlignment.Left,
        TextTruncate = Enum.TextTruncate.AtEnd,
        ClearTextOnFocus = false,
        Parent = InputFrame
    })
    
    local ExecuteBtn = CreateElement("TextButton", {
        Name = "ValidateBtn",
        Size = UDim2.new(0, 72, 1, 0),
        Position = UDim2.new(1, 0, 0, 0),
        AnchorPoint = Vector2.new(1, 0),
        BackgroundColor3 = Color3.fromRGB(255, 255, 255),
        Text = "Validate",
        TextColor3 = Color3.fromRGB(15, 15, 20),
        Font = Enum.Font.GothamMedium,
        TextSize = 12,
        AutoButtonColor = false,
        Parent = InputRow
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 8), Parent = ExecuteBtn })
    
    local ExecuteGradient = CreateElement("UIGradient", {
        Color = ColorSequence.new({
            ColorSequenceKeypoint.new(0, Color3.fromRGB(255, 255, 255)),
            ColorSequenceKeypoint.new(0.5, Color3.fromRGB(200, 200, 210)),
            ColorSequenceKeypoint.new(1, Color3.fromRGB(120, 120, 130))
        }),
        Rotation = 90,
        Parent = ExecuteBtn
    })
    
    CreateElement("UIStroke", {
        Color = Color3.fromRGB(80, 80, 90),
        Thickness = 1,
        Transparency = 0.5,
        Parent = ExecuteBtn
    })
    
    local ActionsRow = CreateElement("Frame", {
        Name = "ActionsRow",
        Size = UDim2.new(1, 0, 0, 32),
        Position = UDim2.new(0, 0, 0, 46),
        BackgroundTransparency = 1,
        Parent = Content
    })
    
    local DiscordBtn = CreateElement("TextButton", {
        Name = "DiscordBtn",
        Size = UDim2.new(0.5, -4, 1, 0),
        BackgroundColor3 = C.Glass,
        BackgroundTransparency = 0.4,
        Text = "Discord",
        TextColor3 = C.TextMuted,
        Font = Enum.Font.Gotham,
        TextSize = 11,
        AutoButtonColor = false,
        Parent = ActionsRow
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 6), Parent = DiscordBtn })
    CreateElement("UIStroke", { Color = C.Border, Thickness = 1, Transparency = 0.7, Parent = DiscordBtn })
    
    local GetKeyBtn = CreateElement("TextButton", {
        Name = "GetKeyBtn",
        Size = UDim2.new(0.5, -4, 1, 0),
        Position = UDim2.new(0.5, 4, 0, 0),
        BackgroundColor3 = C.Glass,
        BackgroundTransparency = 0.4,
        Text = "Get Key",
        TextColor3 = C.TextMuted,
        Font = Enum.Font.Gotham,
        TextSize = 11,
        AutoButtonColor = false,
        Parent = ActionsRow
    })
    CreateElement("UICorner", { CornerRadius = UDim.new(0, 6), Parent = GetKeyBtn })
    CreateElement("UIStroke", { Color = C.Border, Thickness = 1, Transparency = 0.7, Parent = GetKeyBtn })
    
    local function Notify(msg, success)
        local color = success and C.Success or C.Error
        
        local NotifFrame = CreateElement("Frame", {
            Size = UDim2.new(0, 280, 0, 44),
            Position = UDim2.new(1, -16, 1, -16),
            AnchorPoint = Vector2.new(1, 1),
            BackgroundColor3 = C.Background,
            BackgroundTransparency = 0.1,
            Parent = ScreenGui
        })
        CreateElement("UICorner", { CornerRadius = UDim.new(0, 8), Parent = NotifFrame })
        CreateElement("UIStroke", { Color = color, Thickness = 1, Transparency = 0.5, Parent = NotifFrame })
        
        local Indicator = CreateElement("Frame", {
            Size = UDim2.new(0, 3, 0.6, 0),
            Position = UDim2.new(0, 8, 0.5, 0),
            AnchorPoint = Vector2.new(0, 0.5),
            BackgroundColor3 = color,
            BorderSizePixel = 0,
            Parent = NotifFrame
        })
        CreateElement("UICorner", { CornerRadius = UDim.new(0, 2), Parent = Indicator })
        
        CreateElement("TextLabel", {
            Size = UDim2.new(1, -24, 1, 0),
            Position = UDim2.new(0, 20, 0, 0),
            BackgroundTransparency = 1,
            Text = msg,
            TextColor3 = C.Text,
            TextSize = 12,
            Font = Enum.Font.Gotham,
            TextXAlignment = Enum.TextXAlignment.Left,
            TextTruncate = Enum.TextTruncate.AtEnd,
            Parent = NotifFrame
        })
        
        NotifFrame.Position = UDim2.new(1, 100, 1, -16)
        Tween(NotifFrame, {Position = UDim2.new(1, -16, 1, -16)}, 0.3):Play()
        
        task.delay(2.5, function()
            Tween(NotifFrame, {Position = UDim2.new(1, 100, 1, -16)}, 0.25):Play()
            task.wait(0.25)
            NotifFrame:Destroy()
        end)
    end
    
    local function AddHover(btn, normalColor, hoverColor, normalTrans, hoverTrans)
        btn.MouseEnter:Connect(function()
            Tween(btn, {BackgroundColor3 = hoverColor, BackgroundTransparency = hoverTrans or 0}, 0.15):Play()
        end)
        btn.MouseLeave:Connect(function()
            Tween(btn, {BackgroundColor3 = normalColor, BackgroundTransparency = normalTrans or 0}, 0.15):Play()
        end)
    end
    
    ExecuteBtn.MouseEnter:Connect(function()
        Tween(ExecuteGradient, {Rotation = 270}, 0.2):Play()
        Tween(ExecuteBtn, {BackgroundColor3 = Color3.fromRGB(255, 255, 255)}, 0.15):Play()
    end)
    ExecuteBtn.MouseLeave:Connect(function()
        Tween(ExecuteGradient, {Rotation = 90}, 0.2):Play()
    end)
    
    AddHover(DiscordBtn, C.Glass, C.GlassHover, 0.4, 0.2)
    AddHover(GetKeyBtn, C.Glass, C.GlassHover, 0.4, 0.2)
    
    CloseBtn.MouseEnter:Connect(function()
        Tween(CloseBtn, {BackgroundColor3 = C.Error, BackgroundTransparency = 0.2}, 0.15):Play()
        Tween(CloseIcon, {ImageColor3 = C.Text}, 0.15):Play()
    end)
    CloseBtn.MouseLeave:Connect(function()
        Tween(CloseBtn, {BackgroundColor3 = C.Glass, BackgroundTransparency = 0.5}, 0.15):Play()
        Tween(CloseIcon, {ImageColor3 = C.TextMuted}, 0.15):Play()
    end)
    
    KeyInput.Focused:Connect(function()
        Tween(InputStroke, {Color = C.Accent, Transparency = 0}, 0.2):Play()
    end)
    KeyInput.FocusLost:Connect(function()
        Tween(InputStroke, {Color = C.Border, Transparency = 0.6}, 0.2):Play()
    end)
    
    local function ShowModeSelector(onSelect)
        local Overlay = CreateElement("Frame", {
            Name = "ModeOverlay",
            Size = UDim2.new(1, 0, 1, 0),
            BackgroundColor3 = Color3.fromRGB(0, 0, 0),
            BackgroundTransparency = 0.5,
            Parent = ScreenGui
        })
        
        local Modal = CreateElement("Frame", {
            Name = "ModeModal",
            Size = UDim2.new(0, 260, 0, 0),
            Position = UDim2.new(0.5, 0, 0.5, 0),
            AnchorPoint = Vector2.new(0.5, 0.5),
            BackgroundColor3 = C.Background,
            ClipsDescendants = true,
            Parent = ScreenGui
        })
        CreateElement("UICorner", { CornerRadius = UDim.new(0, 10), Parent = Modal })
        CreateElement("UIStroke", { Color = C.Border, Thickness = 1, Transparency = 0.5, Parent = Modal })
        
        local ModalTitle = CreateElement("TextLabel", {
            Size = UDim2.new(1, 0, 0, 36),
            BackgroundTransparency = 1,
            Text = "Select Mode",
            TextColor3 = C.Text,
            TextSize = 13,
            Font = Enum.Font.GothamMedium,
            Parent = Modal
        })
        
        CreateElement("Frame", {
            Size = UDim2.new(1, -24, 0, 1),
            Position = UDim2.new(0, 12, 0, 36),
            BackgroundColor3 = C.Border,
            BackgroundTransparency = 0.5,
            BorderSizePixel = 0,
            Parent = Modal
        })
        
        local BtnContainer = CreateElement("Frame", {
            Size = UDim2.new(1, -24, 0, 70),
            Position = UDim2.new(0, 12, 0, 44),
            BackgroundTransparency = 1,
            Parent = Modal
        })
        
        local NormalBtn = CreateElement("TextButton", {
            Size = UDim2.new(1, 0, 0, 32),
            BackgroundColor3 = C.Glass,
            BackgroundTransparency = 0.3,
            Text = "Normal Mode",
            TextColor3 = C.Text,
            TextSize = 12,
            Font = Enum.Font.Gotham,
            AutoButtonColor = false,
            Parent = BtnContainer
        })
        CreateElement("UICorner", { CornerRadius = UDim.new(0, 6), Parent = NormalBtn })
        CreateElement("UIStroke", { Color = C.Accent, Thickness = 1, Transparency = 0.3, Parent = NormalBtn })
        
        local LiteBtn = CreateElement("TextButton", {
            Size = UDim2.new(1, 0, 0, 32),
            Position = UDim2.new(0, 0, 0, 38),
            BackgroundColor3 = C.Glass,
            BackgroundTransparency = 0.3,
            Text = "Lite Mode",
            TextColor3 = C.TextMuted,
            TextSize = 12,
            Font = Enum.Font.Gotham,
            AutoButtonColor = false,
            Parent = BtnContainer
        })
        CreateElement("UICorner", { CornerRadius = UDim.new(0, 6), Parent = LiteBtn })
        CreateElement("UIStroke", { Color = C.Border, Thickness = 1, Transparency = 0.5, Parent = LiteBtn })
        
        NormalBtn.MouseEnter:Connect(function() Tween(NormalBtn, {BackgroundTransparency = 0.1}, 0.15):Play() end)
        NormalBtn.MouseLeave:Connect(function() Tween(NormalBtn, {BackgroundTransparency = 0.3}, 0.15):Play() end)
        LiteBtn.MouseEnter:Connect(function() Tween(LiteBtn, {BackgroundTransparency = 0.1}, 0.15):Play() end)
        LiteBtn.MouseLeave:Connect(function() Tween(LiteBtn, {BackgroundTransparency = 0.3}, 0.15):Play() end)
        
        NormalBtn.MouseButton1Click:Connect(function()
            Tween(Modal, {Size = UDim2.new(0, 260, 0, 0)}, 0.2):Play()
            Tween(Overlay, {BackgroundTransparency = 1}, 0.2):Play()
            task.wait(0.2)
            Overlay:Destroy()
            Modal:Destroy()
            onSelect(false)
        end)
        
        LiteBtn.MouseButton1Click:Connect(function()
            Tween(Modal, {Size = UDim2.new(0, 260, 0, 0)}, 0.2):Play()
            Tween(Overlay, {BackgroundTransparency = 1}, 0.2):Play()
            task.wait(0.2)
            Overlay:Destroy()
            Modal:Destroy()
            onSelect(true)
        end)
        
        Overlay.BackgroundTransparency = 1
        Tween(Overlay, {BackgroundTransparency = 0.5}, 0.2):Play()
        Tween(Modal, {Size = UDim2.new(0, 260, 0, 130)}, 0.25, Enum.EasingStyle.Back):Play()
    end
    
    local function DoLoadScript(useLite, key)
        local success, loadMsg = LoadGameScript(useLite, key)
        
        -- Always notify and close UI (script was attempted to load)
        Notify(loadMsg, success)
        
        if success then
            StartHeartbeat(key)
        end
        
        -- Always close Key UI after execution attempt
        task.wait(0.8)
        Tween(MainFrame, {Size = UDim2.new(0, 340, 0, 0)}, 0.25):Play()
        task.wait(0.25)
        ScreenGui:Destroy()
    end
    
    ExecuteBtn.MouseButton1Click:Connect(function()
        local key = KeyInput.Text
        if key == "" then
            Notify("Please enter a key", false)
            return
        end
        
        ExecuteBtn.Text = "..."
        Tween(ExecuteGradient, {Rotation = 180}, 0.1):Play()
        
        task.wait(0.3)
        
        local valid, msg, sessionData = ValidateKey(key)
        
        if valid then
            _G.script_key = key

            Notify(msg, true)
            ExecuteBtn.Text = "..."
            
            -- Store session data from API response
            if sessionData then
                Session.MaxAccounts = sessionData.max_accounts or 0
                Session.AccountsUsed = sessionData.accounts_used or 0
                Session.KeyStatus = sessionData.status or "active"
            end
            
            task.wait(0.4)
            
            if HasLiteMode() then
                -- LOGIC CHECK LITE MODE --
                if _G.lite_mode == true then
                     DoLoadScript(true, key)
                else
                    ShowModeSelector(function(useLite)
                        DoLoadScript(useLite, key)
                    end)
                end
            else
                DoLoadScript(false, key)
            end
        else
            Notify(msg, false)
            ExecuteBtn.Text = "Validate"
            Tween(ExecuteGradient, {Rotation = 90}, 0.1):Play()
        end
    end)
    
    CloseBtn.MouseButton1Click:Connect(function()
        StopHeartbeat()
        Tween(MainFrame, {Size = UDim2.new(0, 340, 0, 0)}, 0.2):Play()
        task.wait(0.2)
        ScreenGui:Destroy()
    end)
    
    DiscordBtn.MouseButton1Click:Connect(function()
        setclipboard(Config.DiscordLink)
        Notify("Discord copied!", true)
    end)
    
    GetKeyBtn.MouseButton1Click:Connect(function()
        setclipboard(Config.GetKeyLink)
        Notify("Link copied!", true)
    end)
    
    local dragging, dragInput, dragStart, startPos
    
    Header.InputBegan:Connect(function(input)
        if input.UserInputType == Enum.UserInputType.MouseButton1 then
            dragging = true
            dragStart = input.Position
            startPos = MainFrame.Position
            input.Changed:Connect(function()
                if input.UserInputState == Enum.UserInputState.End then
                    dragging = false
                end
            end)
        end
    end)
    
    Header.InputChanged:Connect(function(input)
        if input.UserInputType == Enum.UserInputType.MouseMovement then
            dragInput = input
        end
    end)
    
    UserInputService.InputChanged:Connect(function(input)
        if input == dragInput and dragging then
            local delta = input.Position - dragStart
            MainFrame.Position = UDim2.new(startPos.X.Scale, startPos.X.Offset + delta.X, startPos.Y.Scale, startPos.Y.Offset + delta.Y)
        end
    end)
    
    Tween(MainFrame, {Size = UDim2.new(0, 340, 0, 145)}, 0.35, Enum.EasingStyle.Back):Play()
end

-- MAIN LOGIC: CHECK AUTO AUTH --
local function CheckAndRun()
    if _G.script_key and #_G.script_key > 0 then
        pcall(function()
            StarterGui:SetCore("SendNotification", {
                Title = Config.ScriptName,
                Text = "Checking key...",
                Duration = 2
            })
        end)

        local valid, msg, sessionData = ValidateKey(_G.script_key)

        if valid then
            pcall(function()
                StarterGui:SetCore("SendNotification", {
                    Title = "Success!",
                    Text = "Loading script...",
                    Duration = 2
                })
            end)
            
            -- Store session data from API response
            if sessionData then
                Session.MaxAccounts = sessionData.max_accounts or 0
                Session.AccountsUsed = sessionData.accounts_used or 0
                Session.KeyStatus = sessionData.status or "active"
            end
            
            local useLite = _G.lite_mode == true
            
            local success, loadMsg = LoadGameScript(useLite, _G.script_key)
            
            if success then
                StartHeartbeat(_G.script_key)
                return 
            end
        else
            warn("Auto-Auth Failed:", msg)
        end
    end
    
    pcall(CreateUI)
end

CheckAndRun()