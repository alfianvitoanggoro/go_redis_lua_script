-- KEYS[1] = q:{user}
-- KEYS[2] = lock:{user}
-- KEYS[3] = ready:wallet

local q     = KEYS[1]
local lock  = KEYS[2]
local ready = KEYS[3]

-- Harus ada lock (kalau tidak, biarkan aplikasi yang repair)
if redis.call('EXISTS', lock) == 0 then
  return -1
end

-- Buang head item yang barusan diproses
redis.call('LPOP', q)

local llen = redis.call('LLEN', q)
if llen > 0 then
  -- Masih ada antrian: tetap locked & tandai siap lagi
  redis.call('LPUSH', ready, q)
  return 1
else
  -- Habis: buka lock
  redis.call('DEL', lock)
  return 0
end
