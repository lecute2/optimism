package store

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-base32"
)

const (
	scoreCacheSize          = 100
	scoreRecordExpiryPeriod = 24 * time.Hour
)

var scoresBase = ds.NewKey("/peers/scores")

type scoreRecord struct {
	PeerScores PeerScores `json:"peerScores"`
	LastUpdate int64      `json:"lastUpdate"` // unix timestamp in seconds
}

func (s *scoreRecord) SetLastUpdated(t time.Time) {
	s.LastUpdate = t.Unix()
}

func (s *scoreRecord) LastUpdated() time.Time {
	return time.Unix(s.LastUpdate, 0)
}

func (s *scoreRecord) MarshalBinary() (data []byte, err error) {
	return serializeScoresV0(*s)
}

func (s *scoreRecord) UnmarshalBinary(data []byte) error {
	r, err := deserializeScoresV0(data)
	if err != nil {
		return err
	}
	*s = r
	return nil
}

type scoreBook struct {
	book *recordsBook[peer.ID, *scoreRecord]
}

func newScoreRecord() *scoreRecord {
	return new(scoreRecord)
}

func peerIDKey(id peer.ID) ds.Key {
	return ds.NewKey(base32.RawStdEncoding.EncodeToString([]byte(id)))
}

func newScoreBook(ctx context.Context, logger log.Logger, clock clock.Clock, store ds.Batching) (*scoreBook, error) {
	book, err := newRecordsBook[peer.ID, *scoreRecord](ctx, logger, clock, store, scoreCacheSize, scoreRecordExpiryPeriod, scoresBase, newScoreRecord, peerIDKey)
	if err != nil {
		return nil, err
	}
	return &scoreBook{book: book}, nil
}

func (d *scoreBook) startGC() {
	d.book.startGC()
}

func (d *scoreBook) GetPeerScores(id peer.ID) (PeerScores, error) {
	record, err := d.book.getRecord(id)
	if err == UnknownRecordErr {
		return PeerScores{}, nil // return zeroed scores by default
	}
	if err != nil {
		return PeerScores{}, err
	}
	return record.PeerScores, nil
}

func (d *scoreBook) SetScore(id peer.ID, diff ScoreDiff) error {
	return d.book.SetRecord(id, diff)
}

func (d *scoreBook) Close() {
	d.book.Close()
}
