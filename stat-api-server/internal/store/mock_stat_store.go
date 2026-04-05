package store

import "github.com/jerry871002/lineup-lab/stat-api-server/internal/api"

type MockStatStore struct {
	TeamData    []api.Team
	BattingData []map[string]any
}

func NewMockStatStore() *MockStatStore {
	return &MockStatStore{
		TeamData: []api.Team{
			{Name: "Team1", Year: 2024},
			{Name: "Team2", Year: 2024},
			{Name: "Team3", Year: 2024},
		},
		BattingData: []map[string]any{
			{"name": "Player1", "at_bat": "50", "hit:": "10"},
			{"name": "Player2", "at_bat": "100", "hit:": "20"},
			{"name": "Player3", "at_bat": "150", "hit:": "30"},
		},
	}
}

func (s *MockStatStore) GetTeams() ([]api.Team, error) {
	return s.TeamData, nil
}

func (s *MockStatStore) GetBattingStat(team string, year int) ([]map[string]any, error) {
	if year == 2024 {
		return s.BattingData, nil
	}
	return nil, nil
}
