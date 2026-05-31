import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { AdminUserItem } from '../../types/api';

interface ConfirmState {
  show: boolean;
  user: AdminUserItem | null;
  newStatus: string;
  action: string;
}

export default function AdminUsers() {
  const [users, setUsers] = useState<AdminUserItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const [confirm, setConfirm] = useState<ConfirmState>({ show: false, user: null, newStatus: '', action: '' });
  const [toast, setToast] = useState<{ show: boolean; message: string; type: 'error' | 'success' }>({ show: false, message: '', type: 'error' });
  const pageSize = 20;

  const fetchUsers = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await adminApi.listUsers(page, pageSize, keyword);
      setUsers(res.data.data || []);
      setTotal(res.data.total || 0);
    } catch {
      setError('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  }, [page, keyword]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const totalPages = Math.ceil(total / pageSize);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    setKeyword(searchInput);
  };

  const handleToggleStatus = (user: AdminUserItem) => {
    const newStatus = user.status === 'active' ? 'disabled' : 'active';
    const action = newStatus === 'disabled' ? '禁用' : '启用';
    setConfirm({ show: true, user, newStatus, action });
  };

  const handleConfirmToggle = async () => {
    if (!confirm.user) return;
    const { user, newStatus, action } = confirm;
    setConfirm({ show: false, user: null, newStatus: '', action: '' });
    setTogglingId(user.id);
    try {
      await adminApi.updateUserStatus(user.id, newStatus);
      setUsers((prev) =>
        prev.map((u) => (u.id === user.id ? { ...u, status: newStatus } : u))
      );
    } catch {
      setToast({ show: true, message: `${action}失败`, type: 'error' });
      setTimeout(() => setToast({ show: false, message: '', type: 'error' }), 3000);
    } finally {
      setTogglingId(null);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">用户管理</h2>
        <button
          onClick={fetchUsers}
          className="text-sm text-purple-600 hover:text-purple-700 font-medium"
        >
          刷新
        </button>
      </div>

      {/* 搜索栏 */}
      <form onSubmit={handleSearch} className="mb-4 flex gap-2">
        <input
          type="text"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="搜索用户名、邮箱或昵称..."
          className="flex-1 max-w-sm rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
        />
        <button
          type="submit"
          className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 transition-colors"
        >
          搜索
        </button>
        {keyword && (
          <button
            type="button"
            onClick={() => {
              setSearchInput('');
              setKeyword('');
              setPage(1);
            }}
            className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
          >
            清除
          </button>
        )}
      </form>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm mb-6">
          {error}
        </div>
      )}

      {loading ? (
        <div className="text-center py-12 text-gray-400">加载中...</div>
      ) : users.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          {keyword ? '未找到匹配的用户' : '暂无用户'}
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-3 font-medium text-gray-500">ID</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">用户名</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">邮箱</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">昵称</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">状态</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">注册时间</th>
                  <th className="text-right py-3 px-3 font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3 px-3 text-gray-400 text-xs">#{user.id}</td>
                    <td className="py-3 px-3">
                      <span className="font-medium text-gray-900">{user.username}</span>
                    </td>
                    <td className="py-3 px-3 text-gray-600 text-xs">{user.email}</td>
                    <td className="py-3 px-3 text-gray-600">{user.display_name || '-'}</td>
                    <td className="py-3 px-3">
                      {user.status === 'active' ? (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700">
                          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                          正常
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700">
                          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 008.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z" clipRule="evenodd" />
                          </svg>
                          已禁用
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-3 text-gray-500 whitespace-nowrap text-xs">{user.created_at}</td>
                    <td className="py-3 px-3 text-right">
                      {user.status === 'active' ? (
                        <button
                          onClick={() => handleToggleStatus(user)}
                          disabled={togglingId === user.id}
                          className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg border border-red-200 text-red-600 hover:bg-red-50 disabled:opacity-50 transition-colors"
                        >
                          {togglingId === user.id ? '处理中...' : '禁用'}
                        </button>
                      ) : (
                        <button
                          onClick={() => handleToggleStatus(user)}
                          disabled={togglingId === user.id}
                          className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-lg border border-green-200 text-green-600 hover:bg-green-50 disabled:opacity-50 transition-colors"
                        >
                          {togglingId === user.id ? '处理中...' : '启用'}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-gray-100">
              <span className="text-sm text-gray-500">
                共 {total} 条记录，第 {page}/{totalPages} 页
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="px-3 py-1.5 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  上一页
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="px-3 py-1.5 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  下一页
                </button>
              </div>
            </div>
          )}
        </>
      )}
      {/* Confirm Modal */}
      {confirm.show && confirm.user && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/40" onClick={() => setConfirm({ show: false, user: null, newStatus: '', action: '' })} />
          <div className="relative bg-white rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 animate-in fade-in zoom-in-95">
            <div className="flex items-start gap-4">
              <div className={`flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center ${confirm.newStatus === 'disabled' ? 'bg-red-100' : 'bg-green-100'}`}>
                {confirm.newStatus === 'disabled' ? (
                  <svg className="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                  </svg>
                ) : (
                  <svg className="w-5 h-5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                )}
              </div>
              <div className="flex-1">
                <h3 className="text-base font-semibold text-gray-900">
                  {confirm.action}用户
                </h3>
                <p className="mt-2 text-sm text-gray-500">
                  确定要{confirm.action}用户 <span className="font-medium text-gray-700">{confirm.user.username}</span> 吗？{confirm.newStatus === 'disabled' && '禁用后该用户将无法登录系统。'}
                </p>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setConfirm({ show: false, user: null, newStatus: '', action: '' })}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-gray-300 text-gray-700 hover:bg-gray-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleConfirmToggle}
                className={`px-4 py-2 text-sm font-medium rounded-lg text-white transition-colors ${
                  confirm.newStatus === 'disabled'
                    ? 'bg-red-600 hover:bg-red-700'
                    : 'bg-green-600 hover:bg-green-700'
                }`}
              >
                确认{confirm.action}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Toast */}
      {toast.show && (
        <div className="fixed top-6 right-6 z-50 animate-in fade-in slide-in-from-top-2">
          <div className={`flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-sm font-medium ${
            toast.type === 'error' ? 'bg-red-600 text-white' : 'bg-green-600 text-white'
          }`}>
            {toast.type === 'error' ? (
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            ) : (
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            )}
            {toast.message}
          </div>
        </div>
      )}
    </div>
  );
}
