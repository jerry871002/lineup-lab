import React, { useState, useEffect } from 'react';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import Lineup from './Lineup.jsx';
import Roster from './Roster.jsx';
import './App.css';

const getApiBaseUrl = (key, fallback) => {
  const value = import.meta.env[key] || fallback;
  return value.replace(/\/$/, '');
};

const statApiBaseUrl = getApiBaseUrl('VITE_STAT_API_BASE_URL', 'http://localhost:8082');
const simulationApiBaseUrl = getApiBaseUrl('VITE_SIMULATION_API_BASE_URL', 'http://localhost:8081');

const App = () => {
  const [teams, setTeams] = useState([]);
  const [selectedTeam, setSelectedTeam] = useState('');
  const [players, setPlayers] = useState([]);
  const [lineup, setLineup] = useState(Array(9).fill(null));
  const [simulationResult, setSimulationResult] = useState(null);

  useEffect(() => {
    fetch(`${statApiBaseUrl}/teams/`)
      .then(response => response.json())
      .then(data => setTeams(data))
      .catch(error => console.error('Error fetching teams:', error));
  }, []);

  useEffect(() => {
    if (selectedTeam) {
      const [name, year] = selectedTeam.split('+');
      fetch(`${statApiBaseUrl}/batting/?team=${name}&year=${year}`)
        .then(response => response.json())
        .then(data => {
          // add AVG, OBP, SLG to the player data
          data.map(player => {
            player.avg = player.hit / player.at_bat;
            player.obp = (player.hit + player.ball_on_base + player.hit_by_pitch) / (player.at_bat + player.ball_on_base + player.hit_by_pitch);
            player.slg = (player.hit + player.double + 2 * player.triple + 3 * player.home_run) / player.at_bat;
            return player;
          });
          setPlayers(data.sort((a, b) => a.name.localeCompare(b.name)));
        })
        .catch(error => console.error('Error fetching players:', error));
    }
  }, [selectedTeam]);

  const onTeamChange = (event) => {
    setSelectedTeam(event.target.value);
    setLineup(Array(9).fill(null));
  };

  const movePlayerToSlot = (player, index) => {
    setSimulationResult(null);

    const newLineup = [...lineup];
    const existingPlayer = newLineup[index];

    // the player is already in the lineup
    const playerIndex = lineup.indexOf(player);
    if (playerIndex !== -1) {
      newLineup[index] = player;
      newLineup[playerIndex] = existingPlayer;
      setLineup(newLineup);
      return;
    }

    // move the existing player back to the roster if there is one
    if (existingPlayer) {
      setPlayers(prevPlayers => 
        [...prevPlayers, existingPlayer].sort((a, b) => a.name.localeCompare(b.name))
      );
    }

    newLineup[index] = player;
    setLineup(newLineup);
    setPlayers(prevPlayers => 
      prevPlayers.filter(p => p.name !== player.name)
    );
  };

  const removePlayerFromSlot = (index) => {
    setSimulationResult(null);
    const player = lineup[index];
    const newLineup = [...lineup];
    newLineup[index] = null;
    setLineup(newLineup);
    setPlayers([...players, player].sort((a, b) => a.name.localeCompare(b.name)));
  };

  const simulateLineup = async () => {
    const response = await fetch(`${simulationApiBaseUrl}/simulate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(lineup),
    });
    const result = await response.json();
    console.log(result);
    setSimulationResult(result);
  };

  return (
    <DndProvider backend={HTML5Backend}>
      <header style={{ textAlign: 'center' }}>
        <h1>Lineup Lab</h1>
      </header>
      <div className="container">
        <div className="card">
          <Lineup
            lineup={lineup}
            movePlayerToSlot={movePlayerToSlot}
            removePlayerFromSlot={removePlayerFromSlot}
            simulateLineup={simulateLineup}
            simulationResult={simulationResult}
          />
        </div>
        <div className="card">
          <Roster players={players} teams={teams} selectedTeam={selectedTeam} onTeamChange={onTeamChange} />
        </div>
      </div>
    </DndProvider>
  );
};

export default App;
