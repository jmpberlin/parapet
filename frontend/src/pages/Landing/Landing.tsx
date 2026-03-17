import { Link } from 'react-router-dom'
import './Landing.scss'

function Landing() {
  return (
    <div className="landing">
      <h1>Parapet</h1>
      <p>Cybersecurity vulnerability monitoring for your repositories.</p>
      <Link to="/dashboard" className="landing__cta">
        Go to Dashboard
      </Link>
    </div>
  )
}

export default Landing
