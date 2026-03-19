import { Link } from 'react-router-dom';
import { useVulnerabilities } from '../../hooks/useVulnerabilities';
import DashboardCard from '../../components/DashboardCard/DashboardCard';
import './Vulnerabilities.scss';

function Vulnerabilities() {
  const { data: vulns, isLoading, error } = useVulnerabilities();

  return (
    <div className='vulnerabilities'>
      <DashboardCard title='Vulnerabilities' count={vulns?.length}>
        {isLoading ? (
          <p className='vulnerabilities__empty'>Loading…</p>
        ) : error ? (
          <p className='vulnerabilities__empty'>Error loading vulnerabilities.</p>
        ) : !vulns || vulns.length === 0 ? (
          <p className='vulnerabilities__empty'>No vulnerabilities found.</p>
        ) : (
          <ul className='vulnerabilities__list'>
            {vulns.map((vuln) => (
              <li key={vuln.id}>
                <Link to={`/vulnerabilities/${vuln.id}`} className='vulnerabilities__item'>
                  <div className='vulnerabilities__info'>
                    <div className='vulnerabilities__header'>
                      <span className='vulnerabilities__cve'>{vuln.cve}</span>
                      <span
                        className={`vulnerabilities__severity vulnerabilities__severity--${vuln.severity.toLowerCase()}`}
                      >
                        {vuln.severity}
                      </span>
                    </div>
                    {vuln.description && (
                      <p className='vulnerabilities__desc'>{vuln.description}</p>
                    )}
                  </div>
                  <span className='vulnerabilities__arrow'>→</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </DashboardCard>
    </div>
  );
}

export default Vulnerabilities;
