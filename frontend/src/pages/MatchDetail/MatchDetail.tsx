import { useParams, Link } from 'react-router-dom';
import { useMatchDetail } from '../../hooks/useRepositories';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import './MatchDetail.scss';

function fmt(iso: string): string {
  const date = new Date(iso);
  // Go's zero time (missing vulnerability) would otherwise render as "1 January 1"
  if (date.getUTCFullYear() === 1) return '—';
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(date);
}

const TIER_LABELS: Record<string, string> = {
  purl: 'PURL',
  namespace: 'namespace',
  normalized_name: 'package name',
  word_token: 'keyword',
};

const CONFIDENCE_LABELS: Record<string, string> = {
  CONFIRMED: 'confirmed',
  WARNING: 'needs review',
  HIGH: 'high confidence',
  MEDIUM: 'medium confidence',
  LOW: 'low confidence',
};

function buildReasoning(
  matchedOn: string,
  vulnId: string,
  depId: string,
  matchedVersion: string,
  versionCheck: string,
  versionRange: string,
): string {
  const tier = TIER_LABELS[matchedOn] ?? matchedOn;
  const lines: string[] = [];

  if (tier) {
    lines.push(`Matched by ${tier}: ${vulnId} ~ ${depId}.`);
  }

  if (versionCheck === 'in_range') {
    lines.push(
      `Version ${matchedVersion || '?'} is inside ${versionRange || 'the affected range'} → confirmed`,
    );
  } else if (versionCheck === 'out_of_range') {
    lines.push(
      `Version ${matchedVersion || '?'} is outside ${versionRange || 'the affected range'} → not affected`,
    );
  } else {
    lines.push(
      `Version check could not be run${versionRange ? ` (range ${versionRange})` : ''} → needs review`,
    );
  }

  return lines.join(' ');
}

function MatchDetail() {
  const { id } = useParams<{ id: string }>();
  const { data: detail, isLoading, error } = useMatchDetail(id ?? '');

  if (isLoading) return <div className='match-detail'><p>Loading…</p></div>;
  if (error) return <div className='match-detail'><p>Error: {error.message}</p></div>;
  if (!detail) return <div className='match-detail'><p>Match not found.</p></div>;

  const { match, dependency, vulnerability, source_articles: sourceArticles } = detail;
  const reasoning = buildReasoning(
    match.matched_on ?? '',
    match.vuln_identifier ?? '',
    match.dep_identifier ?? '',
    match.matched_version,
    match.version_check ?? '',
    vulnerability.affected_technologies[0]?.version_range ?? '',
  );
  const confidenceLabel = match.confidence
    ? (CONFIDENCE_LABELS[match.confidence] ?? match.confidence.toLowerCase())
    : null;

  return (
    <div className='match-detail'>
      <Link to='/repositories' className='match-detail__back'>← Back</Link>
      <div className='match-detail__layout'>

        <DashboardCard>
          <div className='match-detail__header'>
            <span className='match-detail__cve'>{vulnerability.cve || 'Unknown CVE'}</span>
            {vulnerability.severity && (
              <span className={`match-detail__severity match-detail__severity--${vulnerability.severity.toLowerCase()}`}>
                {vulnerability.severity}
              </span>
            )}
            {match.status && (
              <span className={`match-detail__status match-detail__status--${match.status.toLowerCase()}`}>
                {match.status}
              </span>
            )}
          </div>
          <div className='match-detail__divider' />
          {vulnerability.description && (
            <p className='match-detail__desc'>{vulnerability.description}</p>
          )}
          {vulnerability.affected_technologies.length > 0 && (
            <>
              <p className='match-detail__section-label'>Affected technologies</p>
              <ul className='match-detail__tech-list'>
                {vulnerability.affected_technologies.map((tech, i) => (
                  <li key={i} className='match-detail__tech-item'>
                    <span className='match-detail__tech-name'>
                      {tech.name}
                      {tech.purl ? ` · ${tech.purl}` : ''}
                    </span>
                    {tech.version_range && (
                      <span className='match-detail__tech-version'>{tech.version_range}</span>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
          <div className='match-detail__meta-row'>
            <span className='match-detail__meta-label'>Vulnerability published</span>
            <span className='match-detail__meta-value'>{fmt(vulnerability.published_at)}</span>
          </div>
          <div className='match-detail__meta-row'>
            <span className='match-detail__meta-label'>Match detected</span>
            <span className='match-detail__meta-value'>{fmt(match.created_at)}</span>
          </div>
        </DashboardCard>

        <DashboardCard title='Reasoning'>
          <p className='match-detail__reasoning'>{reasoning}</p>
          <div className='match-detail__evidence'>
            <div className='match-detail__meta-row'>
              <span className='match-detail__meta-label'>Matching tier</span>
              <span className='match-detail__meta-value'>
                {TIER_LABELS[match.matched_on ?? ''] ?? match.matched_on ?? '—'}
              </span>
            </div>
            <div className='match-detail__meta-row'>
              <span className='match-detail__meta-label'>Vulnerability identifier</span>
              <span className='match-detail__meta-value'>{match.vuln_identifier || '—'}</span>
            </div>
            <div className='match-detail__meta-row'>
              <span className='match-detail__meta-label'>Dependency identifier</span>
              <span className='match-detail__meta-value'>{match.dep_identifier || '—'}</span>
            </div>
            <div className='match-detail__meta-row'>
              <span className='match-detail__meta-label'>Confidence</span>
              <span className='match-detail__meta-value'>{confidenceLabel ?? '—'}</span>
            </div>
            <div className='match-detail__meta-row'>
              <span className='match-detail__meta-label'>Version check</span>
              <span className='match-detail__meta-value'>
                {(match.version_check ?? 'unknown').replace('_', ' ')}
              </span>
            </div>
          </div>
        </DashboardCard>

        <DashboardCard title='Dependency'>
          <div className='match-detail__meta-row'>
            <span className='match-detail__meta-label'>Name</span>
            <span className='match-detail__meta-value'>{dependency.name}</span>
          </div>
          <div className='match-detail__meta-row'>
            <span className='match-detail__meta-label'>Version</span>
            <span className='match-detail__meta-value'>{dependency.version || '—'}</span>
          </div>
          <div className='match-detail__meta-row'>
            <span className='match-detail__meta-label'>PURL</span>
            <span className='match-detail__meta-value'>{dependency.purl || '—'}</span>
          </div>
        </DashboardCard>

        <DashboardCard title='Source articles' count={sourceArticles.length}>
          {sourceArticles.length === 0 ? (
            <p className='match-detail__empty'>No source articles found.</p>
          ) : (
            <ul className='match-detail__article-list'>
              {sourceArticles.map((article) => (
                <li key={article.id}>
                  <Link to={`/articles/${article.id}`} className='match-detail__article-item'>
                    <div className='match-detail__article-info'>
                      <span className='match-detail__article-meta'>
                        {article.host_domain} · {fmt(article.published_at)}
                      </span>
                      <span className='match-detail__article-headline'>{article.headline}</span>
                    </div>
                    <span className='match-detail__arrow'>→</span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

      </div>
    </div>
  );
}

export default MatchDetail;
