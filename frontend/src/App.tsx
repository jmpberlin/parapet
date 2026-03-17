import { Routes, Route } from 'react-router-dom'
import Landing from './pages/Landing/Landing'
import Dashboard from './pages/Dashboard/Dashboard'
import RepoDetail from './pages/RepoDetail/RepoDetail'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Landing />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/repos/:id" element={<RepoDetail />} />
    </Routes>
  )
}

export default App
