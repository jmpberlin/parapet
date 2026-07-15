import './DashboardCard.scss';

interface DashboardCardProps {
  title?: string;
  count?: number;
  scrollable?: boolean;
  fixedHeight?: boolean;
  className?: string;
  children: React.ReactNode;
}

function DashboardCard({ title, count, scrollable, fixedHeight, className, children }: DashboardCardProps) {
  return (
    <div className={`dashboard-card${fixedHeight ? ' dashboard-card--fixed-height' : ''}${className ? ` ${className}` : ''}`}>
      {title !== undefined && (
        <div className='dashboard-card__header'>
          <span className='dashboard-card__title'>{title}</span>
          {count !== undefined && (
            <span className='dashboard-card__count'>{count}</span>
          )}
        </div>
      )}
      <div
        className={`dashboard-card__body${scrollable ? ' dashboard-card__body--scrollable' : ''}`}
      >
        {children}
      </div>
    </div>
  );
}

export default DashboardCard;
