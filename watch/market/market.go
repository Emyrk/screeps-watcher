package market

import (
	"encoding/json"
	"fmt"
	"time"
)

type StatsResponse struct {
	Ok    int     `json:"ok"`
	Stats []Stats `json:"stats"`
}

type Stats struct {
	ID           string  `json:"_id"`
	ResourceType string  `json:"resourceType"`
	Date         string  `json:"date"`
	Transactions int     `json:"transactions"`
	Volume       int     `json:"volume"`
	AvgPrice     float64 `json:"avgPrice"`
	StddevPrice  float64 `json:"stddevPrice"`
}

func ParseMarketResponse(data []byte) (*StatsResponse, error) {
	resp := &StatsResponse{}
	err := json.Unmarshal(data, resp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if resp.Ok != 1 {
		return nil, fmt.Errorf("non-ok return in response: %d", resp.Ok)
	}
	return resp, nil
}

func (s *Stats) DateTime() (time.Time, error) {
	return time.Parse("2006-01-02", s.Date)
}

func (s *StatsResponse) Today() (*Stats, error) {
	now := time.Now()
	ny, nm, nd := now.Date()
	for _, stat := range s.Stats {
		dt, err := stat.DateTime()
		if err != nil {
			continue
		}

		sy, sm, sd := dt.Date()
		if ny == sy && nm == sm && nd == sd {
			return &stat, nil
		}
	}
	return nil, fmt.Errorf("no stats for today")

}
