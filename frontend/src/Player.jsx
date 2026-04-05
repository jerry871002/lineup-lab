import React from 'react';
import { useDrag } from 'react-dnd';
import { ItemTypes } from './ItemTypes';

const Player = ({ player }) => {
  const [{ isDragging }, drag] = useDrag({
    type: ItemTypes.PLAYER,
    item: player,
    collect: (monitor) => ({
      isDragging: !!monitor.isDragging(),
    }),
  });

  return (
    <div ref={drag} className={`player ${isDragging ? 'is-dragging' : ''}`}>
      <strong>{player.name}</strong> ({player.avg.toFixed(3)} / {player.obp.toFixed(3)} / {player.slg.toFixed(3)})
    </div>
  );
};

export default Player;
