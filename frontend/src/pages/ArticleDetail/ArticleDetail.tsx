import { useParams, Link } from 'react-router-dom';
import { useArticle } from '../../hooks/useArticles';
import { useVulnerabilities } from '../../hooks/useVulnerabilities';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import './ArticleDetail.scss';

function fmt(iso: string): string {
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(iso));
}

function ArticleDetail() {
  const { id } = useParams<{ id: string }>();
  const { data: article, isLoading, error } = useArticle(id ?? '');
  const { data: allVulns } = useVulnerabilities();

  const vulns = allVulns?.filter((v) => v.source_article_ids.includes(id ?? '')) ?? [];

  if (isLoading) return <div className='article-detail'><p>Loading…</p></div>;
  if (error) return <div className='article-detail'><p>Error: {error.message}</p></div>;
  if (!article) return <div className='article-detail'><p>Article not found.</p></div>;

  return (
    <div className='article-detail'>
      <Link to='/articles' className='article-detail__back'>← Back</Link>
      <div className='article-detail__layout'>

        <DashboardCard>
          <p className='article-detail__meta'>
            {article.host_domain}
            {article.author ? ` · ${article.author}` : ''}
            {' · '}{fmt(article.published_at)}
          </p>
          <h4 className='article-detail__headline'>{article.headline}</h4>
          <a
            href={article.source_url}
            target='_blank'
            rel='noopener noreferrer'
            className='article-detail__source'
          >
            View original article →
          </a>
          <div className='article-detail__divider' />
          <p className='article-detail__body'>{article.content_cleaned}</p>
        </DashboardCard>

        <DashboardCard title='Extracted vulnerabilities' count={vulns.length}>
          {vulns.length === 0 ? (
            <p className='article-detail__empty'>No vulnerabilities extracted from this article.</p>
          ) : (
            <ul className='article-detail__vuln-list'>
              {vulns.map((vuln) => (
                <li key={vuln.id} className='article-detail__vuln-item'>
                  <div className='article-detail__vuln-header'>
                    <span className='article-detail__cve'>{vuln.cve}</span>
                    <span
                      className={`article-detail__severity article-detail__severity--${vuln.severity.toLowerCase()}`}
                    >
                      {vuln.severity}
                    </span>
                  </div>
                  {vuln.description && (
                    <p className='article-detail__vuln-desc'>{vuln.description}</p>
                  )}
                  {vuln.affected_technologies.length > 0 && (
                    <ul className='article-detail__tech-list'>
                      {vuln.affected_technologies.map((tech, i) => (
                        <li key={i} className='article-detail__tech-item'>
                          <span className='article-detail__tech-name'>{tech.name}</span>
                          {tech.version_range && (
                            <span className='article-detail__tech-version'>{tech.version_range}</span>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          )}
        </DashboardCard>

      </div>
    </div>
  );
}

export default ArticleDetail;
