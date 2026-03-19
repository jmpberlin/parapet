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
              <Link to='/repositories' onClick={() => setIsOpen(false)}>
                Repositories
              </Link>
            </li>
            <li>
              <Link to='/articles' onClick={() => setIsOpen(false)}>
                Articles
              </Link>
            </li>
            <li>
              <Link to='/vulnerabilities' onClick={() => setIsOpen(false)}>
                Vulnerabilities
              </Link>
            </li>
          </ul>
        </nav>
      )}
    </>
  );
}

export default BurgerMenu;
