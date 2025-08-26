-- KEYS[1] = q:{user}
-- KEYS[2] = lock:{user}
-- KEYS[3] = ready:wallet
-- ARGV[1] = payload JSON (user_id, currency, amount, tx_id)

local q     = KEYS[1]
local lock  = KEYS[2]
local ready = KEYS[3]
local payload = ARGV[1]

-- Enqueue payload ke antrian user
redis.call('RPUSH', q, payload)

-- Jika belum ada lock dan head ada → acquire + tandai siap diproses
if redis.call('EXISTS', lock) == 0 then
  local head = redis.call('LINDEX', q, 0)
  if head ~= false then
    redis.call('SET', lock, '1')
    redis.call('LPUSH', ready, q)  -- dorong queue user ke daftar 'ready'
    return 1
  end
end

return 0
