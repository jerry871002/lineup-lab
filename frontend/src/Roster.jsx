import React from 'react';
import Player from './Player.jsx';

const Roster = ({ players, teams, selectedTeam, onTeamChange }) => {
  return (
    <div>
      <div className="roster-header">
        <h2>Roster</h2>
        <select value={selectedTeam} onChange={onTeamChange} className="dropdown">
          <option value="" disabled>Select a team</option>
          {teams.map((team, index) => (
            <option key={index} value={`${team.name}+${team.year}`}>{team.name} ({team.year})</option>
          ))}
        </select>
      </div>
      {players.map((player, index) => (
        <Player key={index} player={player} />
      ))}
    </div>
  );
};

export default Roster;
