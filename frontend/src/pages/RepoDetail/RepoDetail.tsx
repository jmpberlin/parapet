import { useParams } from 'react-router-dom'
import './RepoDetail.scss'

function RepoDetail() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="repo-detail">
      <h1>Repository Detail</h1>
      <p>Showing details for repository <code>{id}</code></p>
    </div>
  )
}

export default RepoDetail
