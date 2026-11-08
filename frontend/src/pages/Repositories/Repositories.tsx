import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  useRepositories,
  useRepository,
  useRepositoryMatches,
  useRepositoryDependencies,
} from '../../hooks/useRepositories';
import { useArticlesPaginated } from '../../hooks/useArticles';
import { usePipelineStatus, useRunPipeline } from '../../hooks/usePipeline';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import Pagination from '../../components/Pagination/Pagination';
import './Repositories.scss';

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

const ZERO_TIME = '0001-01-01T00:00:00Z';

function Repositories() {
  const { data: repos, isLoading: reposLoading } = useRepositories();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [matchesPage, setMatchesPage] = useState(1);
  const [dependenciesPage, setDependenciesPage] = useState(1);
  const [scannedPage, setScannedPage] = useState(1);

  const effectiveId = selectedId ?? repos?.[0]?.id ?? null;

  const selectRepository = (id: string) => {
    if (id === effectiveId) return;
    setSelectedId(id);
    setMatchesPage(1);
    setDependenciesPage(1);
  };

  const { data: repo, isLoading: repoLoading } = useRepository(effectiveId ?? '');
  const {
    data: matches,
    isLoading: matchesLoading,
    isFetching: matchesFetching,
  } = useRepositoryMatches(effectiveId ?? '', matchesPage);
  const {
    data: dependencies,
    isLoading: dependenciesLoading,
    isFetching: dependenciesFetching,
  } = useRepositoryDependencies(effectiveId ?? '', dependenciesPage);
  const {
    data: scanned,
    isLoading: scannedLoading,
    isFetching: scannedFetching,
  } = useArticlesPaginated(7, scannedPage);
  const { data: pipeline } = usePipelineStatus();
  const { mutate: runPipeline, isPending: isStarting } = useRunPipeline();

  const isRunning = pipeline?.run_in_progress ?? false;
  const hasRun = pipeline?.ran_at && pipeline.ran_at !== ZERO_TIME;

  return (
    <div className='repos'>

      <div className='repos__top-row'>
        <DashboardCard title='Repositories' count={repos?.length}>
          {reposLoading ? (
            <p className='repos__empty'>Loading…</p>
          ) : !repos || repos.length === 0 ? (
            <p className='repos__empty'>No repositories being watched.</p>
          ) : (
            <ul className='repos__list'>
              {repos.map((r) => (
                <li
                  key={r.id}
                  className={`repos__repo-item${effectiveId === r.id ? ' repos__repo-item--active' : ''}`}
                  onClick={() => selectRepository(r.id)}
                >
                  <div className='repos__repo-item-name'>
                    <span className='repos__repo-owner'>{r.owner_name}</span>
                    <span className='repos__repo-separator'>/</span>
                    <span className='repos__repo-reponame'>{r.repository_name}</span>
                  </div>
                  <span className='repos__repo-provider'>{r.git_provider}</span>
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

        <DashboardCard title='Pipeline'>
          <div className='repos__pipeline'>
            <div className='repos__pipeline-actions'>
              {isRunning && (
                <span className='repos__pipeline-running'>
                  <span className='repos__pipeline-dot' />
                  Running…
                </span>
              )}
              <button
                className='repos__pipeline-btn'
                onClick={() => runPipeline()}
                disabled={isRunning || isStarting}
              >
                Run pipeline
              </button>
            </div>

            {hasRun ? (
              <>
                <p className='repos__pipeline-ran-at'>
                  Last run · {fmt(pipeline!.ran_at)}
                </p>
                <div className='repos__pipeline-stats'>
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Articles harvested</span>
                    <span className='repos__value'>{pipeline!.articles_harvested}</span>
                  </div>
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Vulnerabilities extracted</span>
                    <span className='repos__value'>{pipeline!.vulnerabilities_extracted}</span>
                  </div>
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Deps added</span>
                    <span className='repos__value'>{pipeline!.deps_added}</span>
                  </div>
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Deps removed</span>
                    <span className='repos__value'>{pipeline!.deps_removed}</span>
                  </div>
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Matches found</span>
                    <span className='repos__value'>{pipeline!.matches_found}</span>
                  </div>
                </div>
                {pipeline!.errors.length > 0 && (
                  <div className='repos__pipeline-stat'>
                    <span className='repos__label'>Errors</span>
                    <span className='repos__value repos__value--error'>{pipeline!.errors.length}</span>
                  </div>
                )}
              </>
            ) : (
              !isRunning && <p className='repos__empty'>No runs yet.</p>
            )}
          </div>
        </DashboardCard>
      </div>

      {effectiveId && (
        <div
          className={`repos__grid${repoLoading ? ' repos__grid--loading' : ''}`}
        >
          <DashboardCard title='Watching'>
            {repoLoading ? (
              <p className='repos__empty'>Loading…</p>
            ) : repo ? (
              <div className='repos__repo-info'>
                <div className='repos__repo-row'>
                  <span className='repos__label'>Repository</span>
                  <span className='repos__value'>
                    {repo.owner_name}/{repo.repository_name}
                  </span>
                </div>
                <div className='repos__repo-row'>
                  <span className='repos__label'>Provider</span>
                  <span className='repos__value'>{repo.git_provider}</span>
                </div>
                <div className='repos__repo-row'>
                  <span className='repos__label'>Last fetched</span>
                  <span className='repos__value'>
                    {fmt(repo.last_fetched_at)}
                  </span>
                </div>
              </div>
            ) : null}
          </DashboardCard>

          <DashboardCard title='Alerts' count={repo?.match_count} fixedHeight>
            {matchesLoading ? (
              <p className='repos__empty'>Loading…</p>
            ) : !matches || matches.items.length === 0 ? (
              <p className='repos__empty'>
                No vulnerability matches found.
              </p>
            ) : (
              <>
                <ul
                  className={`repos__list${matchesFetching ? ' repos__list--fetching' : ''}`}
                >
                  {matches.items.map((match) => (
                    <li key={match.id} className='repos__match-item'>
                      <Link to={`/matches/${match.id}`} className='repos__match-link'>
                        <span
                          className={`repos__status repos__status--${match.status.toLowerCase()}`}
                        >
                          {match.status}
                        </span>
                        <div className='repos__match-detail'>
                          <span className='repos__match-name'>
                            {match.matched_component}
                          </span>
                          <span className='repos__match-version'>
                            {match.matched_version}
                          </span>
                        </div>
                        <div className='repos__match-badges'>
                          {match.matched_on && (
                            <span className='repos__badge'>{match.matched_on}</span>
                          )}
                          {match.confidence && (
                            <span className='repos__badge'>{match.confidence}</span>
                          )}
                        </div>
                        <span className='repos__match-date repos__date'>
                          {fmt(match.created_at)}
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
                {matchesFetching && (
                  <span className='repos__spinner' aria-label='Loading' />
                )}
                {matches && matches.total_pages > 1 && (
                  <Pagination
                    page={matchesPage}
                    totalPages={matches.total_pages}
                    onChange={setMatchesPage}
                    disabled={matchesFetching}
                  />
                )}
              </>
            )}
          </DashboardCard>

          <DashboardCard
            title='Dependencies'
            count={repo?.dependency_count}
            fixedHeight
          >
            {dependenciesLoading ? (
              <p className='repos__empty'>Loading…</p>
            ) : !dependencies || dependencies.items.length === 0 ? (
              <p className='repos__empty'>No dependencies found.</p>
            ) : (
              <>
                <ul
                  className={`repos__list${dependenciesFetching ? ' repos__list--fetching' : ''}`}
                >
                  {dependencies.items.map((dep) => (
                    <li key={dep.id} className='repos__dep-item'>
                      <span className='repos__dep-name'>{dep.name}</span>
                      <span className='repos__dep-version'>
                        {dep.version}
                      </span>
                    </li>
                  ))}
                </ul>
                {dependenciesFetching && (
                  <span className='repos__spinner' aria-label='Loading' />
                )}
                {dependencies && dependencies.total_pages > 1 && (
                  <Pagination
                    page={dependenciesPage}
                    totalPages={dependencies.total_pages}
                    onChange={setDependenciesPage}
                    disabled={dependenciesFetching}
                  />
                )}
              </>
            )}
          </DashboardCard>

          <DashboardCard title='Scanned' count={scanned?.total} fixedHeight>
            {scannedLoading ? (
              <p className='repos__empty'>Loading…</p>
            ) : !scanned || scanned.items.length === 0 ? (
              <p className='repos__empty'>
                No articles in the last 7 days.
              </p>
            ) : (
              <>
                <ul
                  className={`repos__list${scannedFetching ? ' repos__list--fetching' : ''}`}
                >
                  {scanned.items.map((article) => (
                    <li key={article.id} className='repos__article-item'>
                      <Link
                        to={`/articles/${article.id}`}
                        className='repos__article-link'
                      >
                        {truncate(article.headline)}
                      </Link>
                      <span className='repos__article-meta'>
                        {article.host_domain} · {fmt(article.published_at)}
                      </span>
                    </li>
                  ))}
                </ul>
                {scannedFetching && (
                  <span className='repos__spinner' aria-label='Loading' />
                )}
                {scanned.total_pages > 1 && (
                  <Pagination
                    page={scannedPage}
                    totalPages={scanned.total_pages}
                    onChange={setScannedPage}
                    disabled={scannedFetching}
                  />
                )}
              </>
            )}
          </DashboardCard>
        </div>
      )}
    </div>
  );
}

export default Repositories;
