-- KEYS[1] = q:{user}
-- KEYS[2] = own:{user}
-- ARGV[1] = expected_owner ('' untuk force tanpa cek)

local q   = KEYS[1]
local own = KEYS[2]
local exp = ARGV[1]

local cur = redis.call('GET', own)
if cur and (exp == '' or cur == exp) then
  redis.call('DEL', own)
  return {1, 'forced'}
end
return {0, 'noop'}
