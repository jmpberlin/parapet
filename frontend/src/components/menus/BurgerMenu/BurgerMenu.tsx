import { useState } from 'react';
import { Link } from 'react-router-dom';
import './BurgerMenu.scss';

function BurgerMenu() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        className={`burger-menu ${isOpen ? 'burger-menu--open' : ''}`}
        onClick={() => setIsOpen(!isOpen)}
        aria-label={isOpen ? 'Close menu' : 'Open menu'}
        aria-expanded={isOpen}
      >
        <span className='burger-menu__bar' />
        <span className='burger-menu__bar' />
        <span className='burger-menu__bar' />
      </button>

      {isOpen && (
        <nav className='burger-menu__overlay'>
          <ul>
            <li>
              <Link to='/' onClick={() => setIsOpen(false)}>
                Home
              </Link>
            </li>
            <li>
              <Link to='/dashboard' onClick={() => setIsOpen(false)}>
                Dashboard
              </Link>
            </li>
            <li>
              <Link to='/detail/123' onClick={() => setIsOpen(false)}>
                Details
              </Link>
            </li>
          </ul>
        </nav>
      )}
    </>
  );
}

export default BurgerMenu;
