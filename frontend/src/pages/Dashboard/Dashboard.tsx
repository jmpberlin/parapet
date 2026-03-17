import { useRepository } from '../../hooks/useRepositories';
import './Dashboard.scss';

const REPO_ID = 'xxxxxxxx';

function Dashboard() {
  const { data: repo, isLoading, error } = useRepository(REPO_ID);

  if (isLoading)
    return (
      <div className='dashboard'>
        <p>Loading…</p>
      </div>
    );
  if (error)
    return (
      <div className='dashboard'>
        <p>Error: {error.message}</p>
      </div>
    );
  if (!repo)
    return (
      <div className='dashboard'>
        <p>Repository not found.</p>
      </div>
    );

  return (
    <div className='dashboard'>
      <h1>
        {repo.owner_name}/{repo.repository_name}
      </h1>
      <p>Provider: {repo.git_provider}</p>
      <p>Last fetched: {repo.last_fetched_at ?? 'Never'}</p>

      <h2>Dependencies ({repo.dependencies.length})</h2>
      {repo.dependencies.length === 0 ? (
        <p>No dependencies found.</p>
      ) : (
        <ul>
          {repo.dependencies.map((dep) => (
            <li key={dep.id}>
              {dep.name} @ {dep.version}
            </li>
          ))}
        </ul>
      )}

      <h2>Vulnerability Matches ({repo.matches.length})</h2>
      {repo.matches.length === 0 ? (
        <p>No vulnerability matches found.</p>
      ) : (
        <ul>
          {repo.matches.map((match) => (
            <li key={match.id}>
              {match.component_purl} — Vulnerability {match.vulnerability_id}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default Dashboard;
