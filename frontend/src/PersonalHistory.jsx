import React, { useEffect, useRef, useState } from 'react';
import './PersonalHistory.css';

const PersonalHistory = ({ refreshToken, onSessionExpired }) => {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  const expired = useRef(onSessionExpired);
  expired.current = onSessionExpired;

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError('');
    fetch('/api/simulations', { credentials: 'same-origin', signal: controller.signal })
      .then(async response => {
        if (controller.signal.aborted) return;
        if (response.status === 401) {
          setItems([]);
          expired.current?.();
          return;
        }
        if (!response.ok) throw new Error('Could not load your simulation history.');
        const data = await response.json();
        if (!Array.isArray(data)) throw new Error('The history response could not be read.');
        if (!controller.signal.aborted) setItems(data);
      })
      .catch(failure => {
        if (!controller.signal.aborted) setError(failure.message || 'Could not load your simulation history.');
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [refreshToken, retry]);

  return (
    <section className="personal-history" aria-labelledby="history-heading">
      <h2 id="history-heading">My simulation history</h2>
      <p>Your latest 100 simulations, recorded automatically.</p>
      {loading && <p role="status">Loading history...</p>}
      {error && <div role="alert">{error} <button type="button" onClick={() => setRetry(value => value + 1)}>Retry history</button></div>}
      {!loading && !error && items.length === 0 && <p>Run a simulation to start your history.</p>}
      {!error && items.map(item => (
        <article key={item.id} className="history-run">
          <h3>{item.lineup_name}</h3>
          <p>{item.team} ({item.year}){item.lineup_revision > 0 && <> · Saved revision {item.lineup_revision}</>}</p>
          <p>Average score: <strong>{item.average_score.toFixed(2)}</strong> · Average hits: <strong>{item.average_hits.toFixed(2)}</strong></p>
          <p>{item.sample_count.toLocaleString()} games · {item.simulator_version} · <time dateTime={item.created_at}>{new Date(item.created_at).toLocaleString()}</time></p>
          <details>
            <summary>View recorded batting order</summary>
            <ol>{item.batters.map((batter, index) => <li key={`${item.id}-${index}`}>{batter.name}</li>)}</ol>
          </details>
        </article>
      ))}
    </section>
  );
};

export default PersonalHistory;
