-- KEYS[1]=balance:{user}:{CUR}, KEYS[2]=tx:{user}, KEYS[3]=stream:balance:{user}
-- ARGV[1]=txId, ARGV[2]=amount(int), ARGV[3]=ts_millis, ARGV[4]=meta_json
if redis.call('HEXISTS', KEYS[2], ARGV[1]) == 1 then
  local cur = redis.call('HGET', KEYS[1], 'amount') or '0'
  return {0, cur}
end
local amt = tonumber(ARGV[2])
if not amt or amt <= 0 then
  return {-2, '0'}
end
local newBal = redis.call('HINCRBY', KEYS[1], 'amount', amt)
redis.call('HSET', KEYS[2], ARGV[1], 1)
redis.call('XADD', KEYS[3], '*', 'type','DEPOSIT','txId',ARGV[1],'amount',tostring(amt),'balance',tostring(newBal),'ts',ARGV[3],'meta',ARGV[4])
return {1, tostring(newBal)}
