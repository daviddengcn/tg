package main

import ()

type LogLine struct {
	Time     []byte
	Contents [][]byte
}

// 2014-04-02 01:17:41,885 INFO  [pool-1-thread-1] hbase.MiniZooKeeperCluster(126): Waiting for ZK to come online on port 62278
func canParse(line []byte) bool {
	if len(line) < 30 {
		return false
	}
	// Time: 2014-04-02 01:17:41,885
	if line[10] != ' ' || line[19] != ',' || line[23] != ' ' {
		return false
	}
	return true
}

func parse(lines [][]byte) (ll LogLine, succ bool) {
	if len(lines) == 0 {
		return LogLine{}, false
	}
	line := lines[0]

	if len(line) < 30 {
		return
	}
	// Time: 2014-04-02 01:17:41,885
	if line[10] != ' ' || line[19] != ',' || line[23] != ' ' {
		return
	}
	ll.Time = line[:23]
	line = line[24:]

	ll.Contents = append(make([][]byte, 0, len(lines)), line)
	ll.Contents = append(ll.Contents, lines[1:]...)

	succ = true
	return
}
