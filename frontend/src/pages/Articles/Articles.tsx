import { Link } from 'react-router-dom';
import { useArticles } from '../../hooks/useArticles';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import './Articles.scss';

function fmt(iso: string): string {
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(iso));
}

function Articles() {
  const { data: articles, isLoading, error } = useArticles(365);

  return (
    <div className='articles'>
      <DashboardCard title='Articles' count={articles?.length}>
        {isLoading ? (
          <p className='articles__empty'>Loading…</p>
        ) : error ? (
          <p className='articles__empty'>Error loading articles.</p>
        ) : !articles || articles.length === 0 ? (
          <p className='articles__empty'>No articles found.</p>
        ) : (
          <ul className='articles__list'>
            {articles.map((article) => (
              <li key={article.id}>
                <Link to={`/articles/${article.id}`} className='articles__item'>
                  <div className='articles__info'>
                    <span className='articles__meta'>
                      {article.host_domain} · {fmt(article.published_at)}
                    </span>
                    <span className='articles__headline'>{article.headline}</span>
                  </div>
                  <span className='articles__arrow'>→</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </DashboardCard>
    </div>
  );
}

export default Articles;
