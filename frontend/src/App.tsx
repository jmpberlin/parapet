import { Routes, Route } from 'react-router-dom';
import Landing from './pages/Landing/Landing';
import Dashboard from './pages/Repositories/Repositories';
import Articles from './pages/Articles/Articles';
import ArticleDetail from './pages/ArticleDetail/ArticleDetail';
import Vulnerabilities from './pages/Vulnerabilities/Vulnerabilities';
import VulnerabilityDetail from './pages/VulnerabilityDetail/VulnerabilityDetail';
import Header from './Header/Header';

function App() {
  return (
    <>
      <Header></Header>
      <Routes>
        <Route path='/' element={<Landing />} />
        <Route path='/dashboard' element={<Dashboard />} />
        <Route path='/articles' element={<Articles />} />
        <Route path='/articles/:id' element={<ArticleDetail />} />
        <Route path='/vulnerabilities' element={<Vulnerabilities />} />
        <Route path='/vulnerabilities/:id' element={<VulnerabilityDetail />} />
      </Routes>
    </>
  );
}

export default App;
