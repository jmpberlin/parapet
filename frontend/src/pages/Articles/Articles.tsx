import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useArticlesPaginated } from '../../hooks/useArticles';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import Pagination from '../../components/Pagination/Pagination';
import './Articles.scss';

function fmt(iso: string): string {
  return new Intl.DateTimeFormat('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(iso));
}

function Articles() {
  const [page, setPage] = useState(1);
  const {
    data: articles,
    isLoading,
    isFetching,
    error,
  } = useArticlesPaginated(365, page);

  return (
    <div className='articles'>
      <DashboardCard title='Articles' count={articles?.total}>
        {isLoading ? (
          <p className='articles__empty'>Loading…</p>
        ) : error ? (
          <p className='articles__empty'>Error loading articles.</p>
        ) : !articles || articles.items.length === 0 ? (
          <p className='articles__empty'>No articles found.</p>
        ) : (
          <>
            <ul
              className={`articles__list${isFetching ? ' articles__list--fetching' : ''}`}
            >
              {articles.items.map((article) => (
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
            {articles.total_pages > 1 && (
              <Pagination
                page={articles.page}
                totalPages={articles.total_pages}
                onChange={setPage}
                disabled={isFetching}
              />
            )}
          </>
        )}
      </DashboardCard>
    </div>
  );
}

export default Articles;
