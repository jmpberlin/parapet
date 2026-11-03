import './Pagination.scss';

interface PaginationProps {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
  disabled?: boolean;
}

function Pagination({ page, totalPages, onChange, disabled = false }: PaginationProps) {
  return (
    <div className='pagination'>
      <button
        className='pagination__btn'
        onClick={() => onChange(Math.max(1, page - 1))}
        disabled={disabled || page === 1}
      >
        ← Previous
      </button>
      <span className='pagination__info'>
        Page {page} of {totalPages}
      </span>
      <button
        className='pagination__btn'
        onClick={() => onChange(Math.min(totalPages, page + 1))}
        disabled={disabled || page === totalPages}
      >
        Next →
      </button>
    </div>
  );
}

export default Pagination;
