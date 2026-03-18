import './Landing.scss';
import macintosh from '../../assets/images/macintosh_v2.png';
import nistLogo from '../../assets/images/nist_logo_v1.png';
import hackerNewsLogo from '../../assets/images/the_hacker_news_logo_v1.jpg';
import bleepingComputerLogo from '../../assets/images/bleeping_computer_logo_v1.jpg';

function Landing() {
  return (
    <div className='landing'>
      <div className='landing__hero'>
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
      <div className='landing'>
        <div className='landing__content'>
          <div className='landing__divider'></div>

          <h5 className='landing__section-label'>What is parapet?</h5>

          <p className='landing__body'>
            When I moved into cybersecurity, I picked up a habit that most
            people in the field share: reading the morning security briefings.
            BleepingComputer. Hacker News. The CVE feeds.
          </p>

          <p className='landing__body'>
            But soon I realised, that everytime I read about a vulnerability I
            tried to remember if I was using it in some project I was working
            on. Or if we were using that specific affected version. And how easily
            we could patch if we needed to.
          </p>

          <p className='landing__body'>
            That check is obviously not really feasible in a bigger environment
            or working across different repositories.{' '}
            <em>
              So I started building parapet to do this work for me and alert me
              in case it finds dependencies at risk.
            </em>
          </p>

          <div className='landing__divider'></div>

          <h5 className='landing__section-label'>How it works:</h5>

          <p className='landing__body'>
            Parapet crawls security news pages, uses an LLM to extract
            vulnerabilities and the software components they affect, then
            cross-references those components against the dependencies of your
            GitHub, Bitbucket repository. When there's a match, you integrate
            your favorite way of getting notificated. Slack, Mail, SMS...
          </p>

          <div className='landing__divider'></div>

          <h5 className='landing__section-label'>Underlying technolgy:</h5>

          <p className='landing__body'>
            The backend is written in Go with a clean architecture approach in
            mind— domain, usecase, adapter, repository layers. Each layer
            communicates through interfaces, making it straightforward to swap
            scrapers, LLM providers, or data sources without touching business
            logic. PostgreSQL handles persistence, goose manages migrations, and
            the pipeline runs on an in-process scheduler.
          </p>

          <p className='landing__body'>
            Vulnerability extraction leverages Anthropics Claude API with
            structured tool use — the model reads article content and returns
            typed, validated data. Dependency data comes from GitHub's SBOM
            endpoint, returning package URLs that are matched against extracted
            vulnerability signatures. The frontend is React with TypeScript,
            typed end-to-end from an OpenAPI spec. The whole thing is deployed
            on a Hetzner VPS behind Caddy.
          </p>
          <div className='landing__divider'></div>

          <h5 className='landing__section-label'>Status:</h5>

          <p className='landing__body'>
            Parapet is a portfolio project. It is not a finished product. The
            matching logic is honest about its limitations — version range
            checking is on the roadmap, and NVD integration would make confirmed
            matches significantly more reliable. But the pipeline runs: articles
            get scraped, cleaned, send to an LLM. The extracted vulnerabilities
            are being saved. The pipeline then fetches a list of up to date
            dependencies and stores matches. All functionalities are reliable,
            idempotent and all foud information get's deduplicated and merged,
            so you don't get tons of alarms in case multiple sources report on
            the same vulnerability like react2server or kubernetes
            ingress-nightmare etc.
          </p>
          <div className='landing__divider'></div>

          <h5 className='landing__section-label'>Challenges:</h5>

          <p className='landing__body'>
            Two unresolved challenges are making the LLM produce consistent and
            reliable results and matching those results. The first of those is
            of course always a challange when working with any AI model. I'm
            experimenting with using claudes online research tool, to let it
            confirm it's own results before returning them. Basically
            researching the found dependency at risk and ideally returning a
            correct PURL. This would again make the second challenge way easier
            to solve: matching two pieces of software. Since there are so many
            different ecosystems and naming conventions, it's really not that
            easy to make good guesses on wether two things are the same. Too
            strict matching rules might discard matches which are valid. Too
            fuzzy and loose matching and you'll have an alert every morning.
          </p>

          <div className='landing__links'>
            <p>
              <a href='#' className='landing__link'>
                View on GitHub
              </a>
            </p>
            <p>
              <a href='#' className='landing__link'>
                Open the dashboard →
              </a>
            </p>
          </div>

          <p className='landing__body'>Built by Johannes Polte · 2026</p>
        </div>
      </div>
    </div>
  );
}

export default Landing;
