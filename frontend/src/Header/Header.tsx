import './Header.scss';
import { Link } from 'react-router-dom';
import BurgerMenu from '../components/Menus/BurgerMenu/BurgerMenu';
import logo from '../assets/images/parapet_logo_v1.png';

function Header() {
  return (
    <div className='header-wrapper'>
      <Link to='/' className='title'>
        <img src={logo} alt='Parapet logo' className='header-logo' />
        <h2>Parapet</h2>
      </Link>
      <div className='menu-wrapper'>
        <BurgerMenu></BurgerMenu>
      </div>
    </div>
  );
}

export default Header;
