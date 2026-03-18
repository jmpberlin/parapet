import { Routes, Route } from 'react-router-dom';
import Landing from './pages/Landing/Landing';
import Dashboard from './pages/Dashboard/Dashboard';
import RepoDetail from './pages/RepoDetail/RepoDetail';
import ArticleDetail from './pages/ArticleDetail/ArticleDetail';
import Header from './Header/Header';

function App() {
  return (
    <>
      <Header></Header>
      <Routes>
        <Route path='/' element={<Landing />} />
        <Route path='/dashboard' element={<Dashboard />} />
        <Route path='/repos/:id' element={<RepoDetail />} />
        <Route path='/articles/:id' element={<ArticleDetail />} />
      </Routes>
    </>
  );
}

export default App;
