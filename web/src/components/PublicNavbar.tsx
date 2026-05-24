import { useState, useRef, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

export default function PublicNavbar() {
  const { user, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const isAuthPage = location.pathname === '/login' || location.pathname === '/register';

  const topLinks = [
    { to: '/models', label: '模型' },
    { to: '/arena', label: '操练场' },
    { to: '/docs', label: '文档' },
  ];

  return (
    <nav className="sticky top-0 z-50 bg-white/80 backdrop-blur-md border-b border-gray-200">
      <div className="max-w-[1440px] mx-auto px-6 h-16 flex items-center justify-between">
        {/* 左侧: 品牌 */}
        <Link to="/" className="flex items-center gap-2.5">
          <span className="text-xl font-bold">
            <span className="bg-gradient-to-r from-orange-400 to-red-500 bg-clip-text text-transparent">
              ModelGate
            </span>
          </span>
        </Link>

        {/* 右侧: 导航链接 */}
        <div className="flex items-center gap-6">
          {topLinks.map((link) => (
            <Link
              key={link.to}
              to={link.to}
              className="text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors"
            >
              {link.label}
            </Link>
          ))}

          {isAuthenticated ? (
            /* 头像下拉 */
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setDropdownOpen(!dropdownOpen)}
                className="flex items-center justify-center w-8 h-8 rounded-full bg-gradient-to-br from-orange-400 to-red-500 text-white text-sm font-bold hover:opacity-90 transition-opacity"
              >
                {user?.username?.charAt(0).toUpperCase() || 'U'}
              </button>

              {dropdownOpen && (
                <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-100 py-1 z-50">
                  <div className="px-4 py-2.5 border-b border-gray-100">
                    <p className="text-sm font-medium text-gray-900">{user?.username}</p>
                    <p className="text-xs text-gray-500 mt-0.5 truncate">{user?.email}</p>
                  </div>
                  <Link
                    to="/dashboard"
                    className="block w-full text-left px-4 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
                    onClick={() => setDropdownOpen(false)}
                  >
                    控制台
                  </Link>
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
                  >
                    退出登录
                  </button>
                </div>
              )}
            </div>
          ) : !isAuthPage ? (
            <>
              <Link
                to="/login"
                className="text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors"
              >
                登录
              </Link>
              <Link
                to="/register"
                className="inline-flex items-center justify-center rounded-lg bg-orange-500 px-4 py-2 text-sm font-semibold text-white hover:bg-orange-600 transition-colors shadow-sm"
              >
                免费注册
              </Link>
            </>
          ) : null}
        </div>
      </div>
    </nav>
  );
}
