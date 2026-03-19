import { Link } from 'react-router-dom';
import './Landing.scss';
import macintosh from '../../assets/images/macintosh_v2.png';
import nistLogo from '../../assets/images/nist_logo_v1.png';
import hackerNewsLogo from '../../assets/images/the_hacker_news_logo_v1.jpg';
import bleepingComputerLogo from '../../assets/images/bleeping_computer_logo_v1.jpg';
import DashboardCard from '../../components/DashboardCard/DashboardCard';

function Landing() {
  return (
    <div className='landing'>
      <div className='landing__sections'>
        <DashboardCard className='landing__hero'>
          <div className='landing__hero-content'>
            <div className='landing__mac-wrapper'>
              <img src={macintosh} alt='' className='landing__mac' />
              <div className='landing__screen-text'>
                <span>watching ☑</span>
                <span>analysing ☑</span>
                <span>
                  alerting
                  <span className='landing__typing-dot landing__typing-dot--1'>
                    .
                  </span>
                  <span className='landing__typing-dot landing__typing-dot--2'>
                    .
                  </span>
                  <span className='landing__typing-dot landing__typing-dot--3'>
                    .
                  </span>
                </span>
              </div>
            </div>
            <div className='landing__flow' aria-hidden='true'>
              <span className='landing__stream-dot landing__stream-dot--1' />
              <span className='landing__stream-dot landing__stream-dot--2' />
              <span className='landing__stream-dot landing__stream-dot--3' />
            </div>
            <div className='landing__sources'>
              <img
                src={hackerNewsLogo}
                alt='The Hacker News'
                className='landing__source-logo'
              />
              <img
                src={bleepingComputerLogo}
                alt='Bleeping Computer'
                className='landing__source-logo'
              />
              <img src={nistLogo} alt='NIST' className='landing__source-logo' />
            </div>
          </div>
        </DashboardCard>

        <div className='landing__sections-grid'>
          <DashboardCard title='What is Parapet?'>
            <p className='landing__body'>
              When I moved into cybersecurity, I picked up a habit that most
              people in the field share: reading the morning security briefings.
              BleepingComputer. Hacker News. The CVE feeds.
            </p>
            <p className='landing__body'>
              But soon I realised, that everytime I read about a vulnerability I
              tried to remember if I was using it in some project I was working
              on. Or if we were using that specific affected version. And how
              easily we could patch if we needed to.
            </p>
            <p className='landing__body landing__body--last'>
              That check is obviously not really feasible in a bigger
              environment or working across different repositories.{' '}
              <em>
                So I started building Parapet to do this work for me and alert
                me in case it finds dependencies at risk.
              </em>
            </p>
          </DashboardCard>

          <DashboardCard title='How it works'>
            <p className='landing__body landing__body--last'>
              Parapet crawls security news pages, uses an LLM to extract
              vulnerabilities and the software components they affect, then
              cross-references those components against the dependencies of your
              GitHub or Bitbucket repository. When there's a match, you
              integrate your favourite way of getting notified — Slack, email,
              SMS.
            </p>
          </DashboardCard>

          <DashboardCard title='Underlying Technology'>
            <p className='landing__body'>
              The backend is written in Go with a clean architecture in mind —
              domain, usecase, adapter, repository layers. Each layer
              communicates through interfaces, making it straightforward to swap
              scrapers, LLM providers, or data sources without touching business
              logic. PostgreSQL handles persistence, goose manages migrations,
              and the pipeline runs on an in-process scheduler.
            </p>
            <p className='landing__body landing__body--last'>
              Vulnerability extraction leverages the Claude API with structured
              tool use — the model reads article content and returns typed,
              validated data. Dependency data comes from GitHub's SBOM endpoint,
              returning package URLs matched against extracted vulnerability
              signatures. The frontend is React with TypeScript, typed
              end-to-end from an OpenAPI spec. The whole thing is deployed on a
              Hetzner VPS behind Caddy.
            </p>
          </DashboardCard>

          <DashboardCard title='Status'>
            <p className='landing__body landing__body--last'>
              Parapet is a portfolio project — not a finished product. The
              matching logic is honest about its limitations: version range
              checking is on the roadmap, and NVD integration would make
              confirmed matches significantly more reliable. But the pipeline
              runs. Articles get scraped, cleaned, and sent to an LLM.
              Vulnerabilities are extracted and saved. Dependencies are fetched
              and matches are stored. All data is deduplicated and merged, so
              you won't get duplicate alerts when multiple sources cover the
              same vulnerability.
            </p>
          </DashboardCard>

          <DashboardCard title='Challenges'>
            <p className='landing__body landing__body--last'>
              Two open challenges remain: getting the LLM to produce consistent
              and reliable results, and matching those results to dependencies.
              I'm experimenting with using Claude's web search tool to let it
              verify its own findings before returning them — ideally resolving
              a correct PURL. That would also simplify the second challenge:
              matching two pieces of software across ecosystems and naming
              conventions is genuinely hard. Too strict and you miss valid
              matches. Too loose and you get an alert every morning.
            </p>
          </DashboardCard>
        </div>

        <div className='landing__footer'>
          <div className='landing__footer-links'>
            <a
              href='https://www.github.com/jmpberlin/nightwatch'
              className='landing__link'
              target='new'
            >
              View on GitHub
            </a>
            <Link to='/dashboard' className='landing__link'>
              Open the dashboard →
            </Link>
          </div>
          <p className='landing__footer-credit'>
            Built by Johannes Polte · 2026
          </p>
        </div>
      </div>
    </div>
  );
}

export default Landing;
