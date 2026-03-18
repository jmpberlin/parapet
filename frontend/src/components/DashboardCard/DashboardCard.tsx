import './DashboardCard.scss';

interface DashboardCardProps {
  title: string;
  count?: number;
  scrollable?: boolean;
  children: React.ReactNode;
}

function DashboardCard({ title, count, scrollable, children }: DashboardCardProps) {
  return (
    <div className='dashboard-card'>
      <div className='dashboard-card__header'>
        <span className='dashboard-card__title'>{title}</span>
        {count !== undefined && (
          <span className='dashboard-card__count'>{count}</span>
        )}
      </div>
      <div
        className={`dashboard-card__body${scrollable ? ' dashboard-card__body--scrollable' : ''}`}
      >
        {children}
      </div>
    </div>
  );
}

export default DashboardCard;
