import React from 'react';
import { useDrop } from 'react-dnd';
import { ItemTypes } from './ItemTypes';
import Player from './Player.jsx';
import { IoPersonRemove } from 'react-icons/io5';


const LineupSlot = ({ player, index, movePlayerToSlot, removePlayerFromSlot }) => {
  const [{ isOver }, drop] = useDrop({
    accept: ItemTypes.PLAYER,
    drop: (draggedPlayer) => movePlayerToSlot(draggedPlayer, index),
    collect: (monitor) => ({
      isOver: !!monitor.isOver(),
    }),
  });

  return (
    <div ref={drop} className={`lineup-slot ${isOver ? 'is-over' : ''}`} style={{ display: 'flex', alignItems: 'center' }}>
      <span>{index + 1}.&nbsp;</span>
      {player ? (
        <Player player={player} />
      ) : (
        <div className="empty-slot"></div>
      )}
      {player && (
        <IoPersonRemove onClick={() => removePlayerFromSlot(index)} style={{ marginLeft: 'auto' }} />
      )}
    </div>
  );
};

export default LineupSlot;
