-- KEYS[1] = q:{user}
-- KEYS[2] = own:{user}
-- ARGV[1] = req_id
-- ARGV[2] = ttl_ms

local q   = KEYS[1]
local own = KEYS[2]
local req = ARGV[1]
local ttl = tonumber(ARGV[2])

-- hanya head yang boleh melepas
local head = redis.call('LINDEX', q, 0)
if head ~= req then
  return {0, 'not_head'}
end

-- keluarkan dirinya dari antrian
redis.call('LPOP', q)

-- promosikan next (jika ada)
local next = redis.call('LINDEX', q, 0)
if next then
  redis.call('SET', own, next, 'PX', ttl)
else
  redis.call('DEL', own)
end

return {1, 'released'}
