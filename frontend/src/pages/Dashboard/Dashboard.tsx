import { Link } from 'react-router-dom';
import { useRepository } from '../../hooks/useRepositories';
import { useArticles } from '../../hooks/useArticles';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import './Dashboard.scss';

const REPO_ID = 'd29eb91f-8759-4d22-8563-043171f8daa2';

function fmt(iso: string | null | undefined): string {
  if (!iso) return '—';
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));
}

function truncate(s: string, max = 62): string {
  return s.length > max ? s.slice(0, max) + '…' : s;
}

function Dashboard() {
  const { data: repo, isLoading: repoLoading, error: repoError } = useRepository(REPO_ID);
  const { data: articles, isLoading: articlesLoading } = useArticles(7);

  if (repoLoading) return <div className='dashboard'><p>Loading…</p></div>;
  if (repoError) return <div className='dashboard'><p>Error: {repoError.message}</p></div>;
  if (!repo) return <div className='dashboard'><p>Repository not found.</p></div>;

  return (
    <div className='dashboard'>
      <div className='dashboard__grid'>

        <DashboardCard title='Watching'>
          <div className='dashboard__repo-info'>
            <div className='dashboard__repo-row'>
              <span className='dashboard__label'>Repository</span>
              <span className='dashboard__value'>
                {repo.owner_name}/{repo.repository_name}
              </span>
            </div>
            <div className='dashboard__repo-row'>
              <span className='dashboard__label'>Provider</span>
              <span className='dashboard__value'>{repo.git_provider}</span>
            </div>
            <div className='dashboard__repo-row'>
              <span className='dashboard__label'>Last fetched</span>
              <span className='dashboard__value'>{fmt(repo.last_fetched_at)}</span>
            </div>
          </div>
        </DashboardCard>

        <DashboardCard title='Alerts' count={repo.matches.length}>
          {repo.matches.length === 0 ? (
            <p className='dashboard__empty'>No vulnerability matches found.</p>
          ) : (
            <ul className='dashboard__list'>
              {repo.matches.map((match) => (
                <li key={match.id} className='dashboard__match-item'>
                  <span
                    className={`dashboard__status dashboard__status--${match.status.toLowerCase()}`}
                  >
                    {match.status}
                  </span>
                  <div className='dashboard__match-detail'>
                    <span className='dashboard__match-name'>{match.matched_component}</span>
                    <span className='dashboard__match-version'>{match.matched_version}</span>
                  </div>
                  <span className='dashboard__match-date dashboard__date'>{fmt(match.created_at)}</span>
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

        <DashboardCard title='Dependencies' count={repo.dependencies.length} scrollable>
          {repo.dependencies.length === 0 ? (
            <p className='dashboard__empty'>No dependencies found.</p>
          ) : (
            <ul className='dashboard__list'>
              {repo.dependencies.map((dep) => (
                <li key={dep.id} className='dashboard__dep-item'>
                  <span className='dashboard__dep-name'>{dep.name}</span>
                  <span className='dashboard__dep-version'>{dep.version}</span>
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

        <DashboardCard title='Scanned' count={articles?.length} scrollable>
          {articlesLoading ? (
            <p className='dashboard__empty'>Loading…</p>
          ) : !articles || articles.length === 0 ? (
            <p className='dashboard__empty'>No articles in the last 7 days.</p>
          ) : (
            <ul className='dashboard__list'>
              {articles.map((article) => (
                <li key={article.id} className='dashboard__article-item'>
                  <Link to={`/articles/${article.id}`} className='dashboard__article-link'>
                    {truncate(article.headline)}
                  </Link>
                  <span className='dashboard__article-meta'>
                    {article.host_domain} · {fmt(article.published_at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

      </div>
    </div>
  );
}

export default Dashboard;
