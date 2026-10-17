package knowledge

import (
	"strings"
	"testing"
)

const talk = `Today we talk about consensus in distributed systems. Raft elects a leader among the servers of a cluster. The leader receives every client request and appends it to its log. It then replicates the log entries to the followers. An entry is committed once a majority of servers have stored it. If the leader crashes, the followers stop hearing its heartbeats. After an election timeout a follower becomes a candidate and asks for votes. A candidate wins the election with votes from a majority of the cluster. Terms number the elections, so a stale leader can always be recognised. Logs that disagree are repaired by the new leader, entry by entry. Safety means that a committed entry is never lost by a later leader. My cat likes to sit on the keyboard while I record these videos. Snapshots compact the log so it does not grow forever. Membership changes use joint consensus to move between configurations. Clients find the leader by asking any server, which redirects them. Reads can be served by the leader after it confirms it still holds a majority. Thanks for watching and see you next week for Paxos. Please like and subscribe to the channel if this helped you. The leader election and log replication are the heart of Raft. Raft was designed to be easier to understand than Paxos. Its paper describes leader election, log replication, and safety separately. Many systems such as etcd and Consul use Raft in production. A majority of three servers survives the loss of one server. A cluster of five servers survives the loss of two servers.`

// A talk becomes five to ten of its own sentences, in the order they were
// said, about what it says rather than the asides.
func TestSummariseKeepsTheCentralSentencesInOrder(t *testing.T) {
	lines := Summarise(talk, 5, 10)
	if len(lines) < 5 || len(lines) > 10 {
		t.Fatalf("%d lines: %q", len(lines), lines)
	}
	at := -1
	for _, l := range lines {
		i := strings.Index(talk, l)
		if i < 0 || i <= at {
			t.Fatalf("%q is not the talk's next sentence", l)
		}
		at = i
	}
	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "leader") || !strings.Contains(joined, "majority") {
		t.Errorf("the summary misses the subject: %q", lines)
	}
	if strings.Contains(joined, "My cat") || strings.Contains(joined, "subscribe") {
		t.Errorf("the summary keeps an aside: %q", lines)
	}
}

// Captions without punctuation are cut into runs of words.
func TestSummariseReadsUnpunctuatedCaptions(t *testing.T) {
	captions := strings.ReplaceAll(strings.ReplaceAll(talk, ".", ""), ",", "")
	lines := Summarise(captions, 5, 10)
	if len(lines) < 4 || len(lines) > 10 {
		t.Fatalf("%d lines", len(lines))
	}
	for _, l := range lines {
		if n := len(strings.Fields(l)); n > 25 {
			t.Errorf("a line of %d words", n)
		}
	}
	if got := Summarise("Too short to say much.", 5, 10); len(got) > 1 {
		t.Errorf("short text = %q", got)
	}
}

func TestKeyPhrasesBetweenFiveAndTen(t *testing.T) {
	k := KeyPhrasesBetween(talk, 5, 10)
	if len(k) < 5 || len(k) > 10 {
		t.Fatalf("%d keywords: %q", len(k), k)
	}
	if !strings.Contains(strings.ToLower(strings.Join(k, " ")), "leader") {
		t.Errorf("keywords = %q", k)
	}
	if k := KeyPhrasesBetween("Raft leader. Raft log. Raft votes. Raft terms. Raft safety.", 5, 10); len(k) < 5 {
		t.Errorf("a short text still gets five: %q", k)
	}
}
