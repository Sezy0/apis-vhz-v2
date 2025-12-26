--[[
    Sync Rare Fish (Mythic + SECRET) to VinzHub API
    Generates 20 of each rare fish and syncs to API
    
    Target User: 2884249490
]]

local HttpService = game:GetService("HttpService")

--------------------------------------------------------------------------------
-- CONFIGURATION
--------------------------------------------------------------------------------

local Config = {
    APIBase = "https://sandbox.vinzhub.com/api/v1",
    APIKey = "vinzhub_sk_live_8f7g6h5j4k3l2m1n0p9o",
    RobloxUserId = "2884249490",
    FishPerTypeMin = 20,  -- Minimum ikan per jenis
    FishPerTypeMax = 40,  -- Maximum ikan per jenis
}

--------------------------------------------------------------------------------
-- RARE FISH DATA (from logs - Mythic T6 + SECRET T7)
--------------------------------------------------------------------------------

local rareFishData = {
    -- SECRET FISH (Tier 7) - Sorted by price
    {Id = 342, Name = "Bloodmoon Whale", Tier = 7, SellPrice = 540000, Icon = "https://tr.rbxcdn.com/180DAY-b4cbfa058b1b550cd3ccefa0ccf47360/420/420/Image/Png/noFilter"},
    {Id = 448, Name = "1x1x1x1 Comet Shark", Tier = 7, SellPrice = 444440, Icon = "https://tr.rbxcdn.com/180DAY-9948eeb88ebf29a61a0423e1c22e77da/420/420/Image/Png/noFilter"},
    {Id = 269, Name = "Elshark Gran Maja", Tier = 7, SellPrice = 440000, Icon = "https://tr.rbxcdn.com/180DAY-86cb026448324f7b7bbf6f78e410ae7d/420/420/Image/Png/noFilter"},
    {Id = 539, Name = "Icebreaker Whale", Tier = 7, SellPrice = 400000, Icon = "https://tr.rbxcdn.com/180DAY-d749a0f05df7f029a319b3571601d91e/420/420/Image/Png/noFilter"},
    {Id = 319, Name = "Zombie Megalodon", Tier = 7, SellPrice = 375000, Icon = "https://tr.rbxcdn.com/180DAY-749e1b19f00c8c2e2ea7c53a69ff981b/420/420/Image/Png/noFilter"},
    {Id = 468, Name = "Strawberry Choc Megalodon", Tier = 7, SellPrice = 375000, Icon = "https://tr.rbxcdn.com/180DAY-276df1ba1ca9734d4b71536b37fab1a8/420/420/Image/Png/noFilter"},
    {Id = 427, Name = "ElRetro Gran Maja", Tier = 7, SellPrice = 360000, Icon = "https://tr.rbxcdn.com/180DAY-a124fb3fee23c00065ce18cb23cda596/420/420/Image/Png/noFilter"},
    {Id = 226, Name = "Megalodon", Tier = 7, SellPrice = 355000, Icon = "https://tr.rbxcdn.com/180DAY-4f77aa0bcdcfa908e42bad55c2544b79/420/420/Image/Png/noFilter"},
    {Id = 228, Name = "Lochness Monster", Tier = 7, SellPrice = 330000, Icon = "https://tr.rbxcdn.com/180DAY-e5870f13dcd954fb9735530fdefeb176/420/420/Image/Png/noFilter"},
    {Id = 159, Name = "Robot Kraken", Tier = 7, SellPrice = 327500, Icon = "https://tr.rbxcdn.com/180DAY-489ffa039a70e41ad60d70997ada55ed/420/420/Image/Png/noFilter"},
    {Id = 509, Name = "Winter Frost Shark", Tier = 7, SellPrice = 320000, Icon = "https://tr.rbxcdn.com/180DAY-04ee420c8d36c89ec8be5edc0e72e490/420/420/Image/Png/noFilter"},
    {Id = 225, Name = "Scare", Tier = 7, SellPrice = 280000, Icon = "https://tr.rbxcdn.com/180DAY-681f78926850a4c4fa9c784bd61ea392/420/420/Image/Png/noFilter"},
    {Id = 145, Name = "Worm Fish", Tier = 7, SellPrice = 280000, Icon = "https://tr.rbxcdn.com/180DAY-7299ff7cfe763fc20a4ad726678c8059/420/420/Image/Png/noFilter"},
    {Id = 295, Name = "Ancient Whale", Tier = 7, SellPrice = 270000, Icon = "https://tr.rbxcdn.com/180DAY-2b9cae3ec359b825fe2fd8e66ef06245/420/420/Image/Png/noFilter"},
    {Id = 293, Name = "Bone Whale", Tier = 7, SellPrice = 255000, Icon = "https://tr.rbxcdn.com/180DAY-308e892131ab2778802ada80f0474cff/420/420/Image/Png/noFilter"},
    {Id = 206, Name = "Monster Shark", Tier = 7, SellPrice = 245000, Icon = "https://tr.rbxcdn.com/180DAY-64489f77e3319c5639d380d34ad871a5/420/420/Image/Png/noFilter"},
    {Id = 200, Name = "Orca", Tier = 7, SellPrice = 231500, Icon = "https://tr.rbxcdn.com/180DAY-e3d4495262428ed404619d8e6f16d6ec/420/420/Image/Png/noFilter"},
    {Id = 292, Name = "King Jelly", Tier = 7, SellPrice = 225000, Icon = "https://tr.rbxcdn.com/180DAY-92fc8bfa788c39623c28f22b0706adc0/420/420/Image/Png/noFilter"},
    {Id = 450, Name = "Depthseeker Ray", Tier = 7, SellPrice = 220000, Icon = "https://tr.rbxcdn.com/180DAY-40969bf064ea5e0893d78709820617d3/420/420/Image/Png/noFilter"},
    {Id = 187, Name = "Queen Crab", Tier = 7, SellPrice = 218500, Icon = "https://tr.rbxcdn.com/180DAY-75f017dc45d893c85ae39873590042b1/420/420/Image/Png/noFilter"},
    {Id = 176, Name = "Ghost Worm Fish", Tier = 7, SellPrice = 195000, Icon = "https://tr.rbxcdn.com/180DAY-fd55b0cb0aceba27ab9e7ad2dc43a18e/420/420/Image/Png/noFilter"},
    {Id = 99, Name = "Great Christmas Whale", Tier = 7, SellPrice = 195000, Icon = "https://tr.rbxcdn.com/180DAY-c53826109548afdbbef84999d9251ad2/420/420/Image/Png/noFilter"},
    {Id = 359, Name = "Gladiator Shark", Tier = 7, SellPrice = 190000, Icon = "https://tr.rbxcdn.com/180DAY-4c8efbc434010eb8503b1eef5f327fe6/420/420/Image/Png/noFilter"},
    {Id = 272, Name = "Mosasaur Shark", Tier = 7, SellPrice = 180000, Icon = "https://tr.rbxcdn.com/180DAY-bbdfb5e8578a81214175a40239d81d96/420/420/Image/Png/noFilter"},
    {Id = 141, Name = "Great Whale", Tier = 7, SellPrice = 180000, Icon = "https://tr.rbxcdn.com/180DAY-427f3f35097642ce11568366678b16af/420/420/Image/Png/noFilter"},
    {Id = 518, Name = "Emerald Winter Whale", Tier = 7, SellPrice = 175000, Icon = "https://tr.rbxcdn.com/180DAY-e4ede90604b1c8f971bccf9ea3c22d61/420/420/Image/Png/noFilter"},
    {Id = 156, Name = "Giant Squid", Tier = 7, SellPrice = 162300, Icon = "https://tr.rbxcdn.com/180DAY-ad8d392c5e6afe4cf421b5d1071b9a5e/420/420/Image/Png/noFilter"},
    {Id = 195, Name = "Crystal Crab", Tier = 7, SellPrice = 162000, Icon = "https://tr.rbxcdn.com/180DAY-644fdd8d9b29c3c4da83e170b9b3eab0/420/420/Image/Png/noFilter"},
    {Id = 445, Name = "1x1x1x1 Shark", Tier = 7, SellPrice = 150000, Icon = "https://tr.rbxcdn.com/180DAY-23999978ceaa0ab507d63c7c89282b47/420/420/Image/Png/noFilter"},
    {Id = 519, Name = "Krampus Shark", Tier = 7, SellPrice = 145000, Icon = "https://tr.rbxcdn.com/180DAY-864b097a096fd2c3683aa302f05199fa/420/420/Image/Png/noFilter"},
    {Id = 339, Name = "Skeleton Narwhal", Tier = 7, SellPrice = 135000, Icon = "https://tr.rbxcdn.com/180DAY-c7d7f5f265fb857bfe91cc9692747dac/420/420/Image/Png/noFilter"},
    {Id = 83, Name = "Ghost Shark", Tier = 7, SellPrice = 125000, Icon = "https://tr.rbxcdn.com/180DAY-6821cd73df0603ad9d833fd93eca7558/420/420/Image/Png/noFilter"},
    {Id = 379, Name = "Cryoshade Glider", Tier = 7, SellPrice = 120000, Icon = "https://tr.rbxcdn.com/180DAY-0ce44b25092b07e5431211628c211d5c/420/420/Image/Png/noFilter"},
    {Id = 136, Name = "Frostborn Shark", Tier = 7, SellPrice = 100000, Icon = "https://tr.rbxcdn.com/180DAY-95e31005ee70d0dd081590e066e2d8a8/420/420/Image/Png/noFilter"},
    {Id = 345, Name = "Ancient Lochness Monster", Tier = 7, SellPrice = 100000, Icon = "https://tr.rbxcdn.com/180DAY-7263c3f92e0b0384f3bd09f1bbb1ca7c/420/420/Image/Png/noFilter"},
    {Id = 82, Name = "Blob Shark", Tier = 7, SellPrice = 98000, Icon = "https://tr.rbxcdn.com/180DAY-de341f23e8add220fefbe2db159fd9ec/420/420/Image/Png/noFilter"},
    {Id = 201, Name = "Eerie Shark", Tier = 7, SellPrice = 92500, Icon = "https://tr.rbxcdn.com/180DAY-b24c218862625f0b1c7a8c84b161cd6d/420/420/Image/Png/noFilter"},
    {Id = 218, Name = "Thin Armor Shark", Tier = 7, SellPrice = 91000, Icon = "https://tr.rbxcdn.com/180DAY-11649b435ccc75c49b95e85f5e3d6011/420/420/Image/Png/noFilter"},
    {Id = 302, Name = "Dead Zombie Shark", Tier = 7, SellPrice = 66000, Icon = "https://tr.rbxcdn.com/180DAY-3171321da5dc01cb79a90621e56446e4/420/420/Image/Png/noFilter"},
    {Id = 297, Name = "Zombie Shark", Tier = 7, SellPrice = 66000, Icon = "https://tr.rbxcdn.com/180DAY-b14df5d94402e2626876f48525c838d3/420/420/Image/Png/noFilter"},
    {Id = 341, Name = "Talon Serpent", Tier = 7, SellPrice = 50000, Icon = "https://tr.rbxcdn.com/180DAY-941fe7a348c160449eec612d16c2ec0c/420/420/Image/Png/noFilter"},
    {Id = 340, Name = "Wild Serpent", Tier = 7, SellPrice = 50000, Icon = "https://tr.rbxcdn.com/180DAY-7a67cc1042a3d44268057c1c0bac2c13/420/420/Image/Png/noFilter"},
    
    -- MYTHIC FISH (Tier 6) - Top ones
    {Id = 158, Name = "King Crab", Tier = 6, SellPrice = 218500, Icon = "https://tr.rbxcdn.com/180DAY-759396ad59e6d31e9d6bf6a025288e4b/420/420/Image/Png/noFilter"},
    {Id = 248, Name = "Panther Eel", Tier = 6, SellPrice = 151500, Icon = "https://tr.rbxcdn.com/180DAY-7909d18f78c0649ac89c3282ee920696/420/420/Image/Png/noFilter"},
    {Id = 424, Name = "Frostbreaker Whale", Tier = 6, SellPrice = 145000, Icon = "https://tr.rbxcdn.com/180DAY-ed54753f7f571f975ca1d7065012800c/420/420/Image/Png/noFilter"},
    {Id = 355, Name = "Flatheaded Whale Shark", Tier = 6, SellPrice = 140000, Icon = "https://tr.rbxcdn.com/180DAY-c6727b35793a7a7cb6a1a249c3edaa3e/420/420/Image/Png/noFilter"},
    {Id = 352, Name = "Cavern Dweller", Tier = 6, SellPrice = 135000, Icon = "https://tr.rbxcdn.com/180DAY-30557ab36397eb18c5b892a583c374d7/420/420/Image/Png/noFilter"},
    {Id = 240, Name = "Magma Shark", Tier = 6, SellPrice = 115500, Icon = "https://tr.rbxcdn.com/180DAY-890b267e5dacdd20b5564118847109a3/420/420/Image/Png/noFilter"},
    {Id = 372, Name = "Runic Squid", Tier = 6, SellPrice = 114000, Icon = "https://tr.rbxcdn.com/180DAY-0388d0df5b5a7c6841b432c9249447dd/420/420/Image/Png/noFilter"},
    {Id = 367, Name = "Primordial Octopus", Tier = 6, SellPrice = 105000, Icon = "https://tr.rbxcdn.com/180DAY-0862650c4b3a47803f935a9438d6303a/420/420/Image/Png/noFilter"},
    {Id = 336, Name = "Hammerhead Mummy", Tier = 6, SellPrice = 100000, Icon = "https://tr.rbxcdn.com/180DAY-8bd199b2359239a309f59d5b2ce26a5e/420/420/Image/Png/noFilter"},
    {Id = 380, Name = "Plasma Serpent", Tier = 6, SellPrice = 98000, Icon = "https://tr.rbxcdn.com/180DAY-35165b79deef0e6438a21bacf94e288c/420/420/Image/Png/noFilter"},
}

--------------------------------------------------------------------------------
-- GENERATE UUID
--------------------------------------------------------------------------------

local function generateUUID()
    local template = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'
    return string.gsub(template, '[xy]', function(c)
        local v = (c == 'x') and math.random(0, 0xf) or math.random(8, 0xb)
        return string.format('%x', v)
    end)
end

--------------------------------------------------------------------------------
-- GENERATE RANDOM WEIGHT
--------------------------------------------------------------------------------

local function generateWeight(tier)
    -- Higher tier = bigger fish
    local baseWeight = tier * 50
    local variation = math.random(1, 100)
    return baseWeight + variation + math.random() * 50
end

--------------------------------------------------------------------------------
-- BUILD INVENTORY DATA
--------------------------------------------------------------------------------

local function buildInventoryData()
    local data = {
        player = {
            user_id = Config.RobloxUserId,
            username = "VinzHub_Demo",
            display_name = "VinzHub Demo"
        },
        fish = {},
        rods = {},
        baits = {},
        potions = {},
        stones = {},
        stats = {
            coins = 999999999,
            level = 100,
        },
        synced_at = os.time()
    }
    
    print("========================================")
    print("🐟 GENERATING RARE FISH INVENTORY")
    print("========================================")
    print(string.format("Fish Types: %d", #rareFishData))
    print(string.format("Fish Per Type: %d - %d (random)", Config.FishPerTypeMin, Config.FishPerTypeMax))
    print("========================================")
    
    local totalFish = 0
    
    for _, fishType in ipairs(rareFishData) do
        -- Random count between min and max
        local fishCount = math.random(Config.FishPerTypeMin, Config.FishPerTypeMax)
        
        for i = 1, fishCount do
            local weight = generateWeight(fishType.Tier)
            local fish = {
                uuid = generateUUID(),
                fish_id = fishType.Id,
                name = fishType.Name,
                tier = fishType.Tier,
                icon = fishType.Icon,
                price = fishType.SellPrice,
                favorited = (i == 1), -- Favorite the first one of each type
                is_shiny = (i <= 2),  -- First 2 are shiny
                metadata = {
                    Weight = weight,
                    CaughtAt = os.time() - math.random(0, 86400 * 30), -- Random time in last 30 days
                },
                variant_id = nil,
                quantity = 1
            }
            
            -- Add some random mutations
            local mutations = {"Golden", "Albino", "Shadow", "Cosmic", "Rainbow", "Crystal"}
            if math.random(1, 10) <= 2 then -- 20% chance of mutation
                fish.variant_id = mutations[math.random(1, #mutations)]
            end
            
            table.insert(data.fish, fish)
        end
        totalFish = totalFish + fishCount
        print(string.format("✅ Generated %d x %s", fishCount, fishType.Name))
    end
    
    print("========================================")
    print(string.format("📦 Total Fish Generated: %d", totalFish))
    print("========================================")
    
    return data
end

--------------------------------------------------------------------------------
-- SYNC TO API
--------------------------------------------------------------------------------

local function syncToAPI()
    print("\n")
    print("========================================")
    print("🚀 SYNCING TO VINZHUB API")
    print("========================================")
    print(string.format("API: %s", Config.APIBase))
    print(string.format("User ID: %s", Config.RobloxUserId))
    print("========================================")
    
    local data = buildInventoryData()
    local jsonBody = HttpService:JSONEncode(data)
    
    local url = Config.APIBase .. "/inventory/" .. Config.RobloxUserId .. "/sync"
    
    print(string.format("📤 Sending %d bytes to API...", #jsonBody))
    
    -- Find HTTP function
    local httpFunc = (syn and syn.request) 
        or request 
        or http_request 
        or (http and http.request)
    
    if not httpFunc then
        warn("❌ No HTTP function available!")
        return false
    end
    
    local success, response = pcall(httpFunc, {
        Url = url,
        Method = "POST",
        Headers = {
            ["Content-Type"] = "application/json",
            ["X-API-Key"] = Config.APIKey
        },
        Body = jsonBody
    })
    
    if not success then
        warn("❌ Request failed:", response)
        return false
    end
    
    print("========================================")
    print(string.format("📥 Response Status: %d", response.StatusCode))
    
    if response.StatusCode == 200 then
        print("✅ SYNC SUCCESSFUL!")
        print("========================================")
        print("📊 SUMMARY:")
        print(string.format("   - Fish Types: %d", #rareFishData))
        print(string.format("   - Fish Per Type: %d", Config.FishPerType))
        print(string.format("   - Total Fish: %d", #data.fish))
        print(string.format("   - User ID: %s", Config.RobloxUserId))
        print("========================================")
        print("🎉 Check your dashboard at sandbox.vinzhub.com!")
        print("========================================")
        return true
    else
        warn("❌ Sync failed:", response.Body)
        return false
    end
end

-- Run it!
syncToAPI()
