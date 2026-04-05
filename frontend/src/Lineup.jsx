import React from 'react';
import LineupSlot from './LineupSlot.jsx';

const Lineup = ({ lineup, movePlayerToSlot, removePlayerFromSlot, simulateLineup, simulationResult }) => {
  return (
    <div>
      <h2>Lineup</h2>
      {lineup.map((player, index) => (
        <LineupSlot
          key={index}
          index={index}
          player={player}
          movePlayerToSlot={movePlayerToSlot}
          removePlayerFromSlot={removePlayerFromSlot}
        />
      ))}
      <div className="simulate-results">
        <button className="simulate-button" onClick={simulateLineup}>Simulate</button>
        {simulationResult && (
          <div>
            <p>Average Score: <strong>{simulationResult.average_score.toFixed(2)}</strong></p>
            <p>Average Hits: <strong>{simulationResult.average_hits.toFixed(2)}</strong></p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Lineup;
