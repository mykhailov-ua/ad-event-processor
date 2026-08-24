package platformconfig

import "ad-event-processor/pkg/runtimepaths"

const (
	RedisShardCountAppliance = 4
	RedisShardCountInfra     = 6
)

func RedisAddrsUDS(shardCount int) string {
	if shardCount <= 0 {
		shardCount = RedisShardCountAppliance
	}
	addrs := make([]string, shardCount)
	for i := 0; i < shardCount; i++ {
		addrs[i] = runtimepaths.RedisSocket(i)
	}
	out := ""
	for i, addr := range addrs {
		if i > 0 {
			out += ","
		}
		out += addr
	}
	return out
}

func RedisAddrsForProfile(profile string) string {
	switch profile {
	case ProfileSingleVPS:
		return RedisAddrsUDS(RedisShardCountAppliance)
	default:
		return RedisAddrsUDS(RedisShardCountAppliance)
	}
}

func RedisShardCountForAddrs(addrs string) int {
	if addrs == "" {
		return 0
	}
	count := 0
	part := ""
	for i := 0; i <= len(addrs); i++ {
		if i == len(addrs) || addrs[i] == ',' {
			if part != "" {
				count++
			}
			part = ""
			continue
		}
		if addrs[i] != ' ' {
			part += string(addrs[i])
		}
	}
	return count
}
