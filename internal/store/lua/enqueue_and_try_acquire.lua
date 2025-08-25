-- KEYS[1] = q:{user}
-- KEYS[2] = own:{user}
-- ARGV[1] = req_id
-- ARGV[2] = ttl_ms

local q   = KEYS[1]
local own = KEYS[2]
local req = ARGV[1]
local ttl = tonumber(ARGV[2])

-- idempotent enqueue: hanya push kalau belum ada
if redis.call('LPOS', q, req) == false then
  redis.call('RPUSH', q, req)
end

-- jika head dan belum ada owner, ambil giliran
local head = redis.call('LINDEX', q, 0)
if head == req and (redis.call('EXISTS', own) == 0) then
  redis.call('SET', own, req, 'PX', ttl)
  return {1, 'acquired'}
end
return {0, 'queued'}
