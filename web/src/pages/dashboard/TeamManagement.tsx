import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import * as teamApi from '../../api/team';
import type { Team } from '../../types/api';

export default function TeamManagement() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Create modal
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  const fetchTeams = useCallback(async () => {
    try {
      setLoading(true);
      const res = await teamApi.getTeams();
      setTeams(res.data.data || []);
    } catch {
      setError('获取团队列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchTeams(); }, [fetchTeams]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateError('');
    if (!newName) {
      setCreateError('请填写团队名称');
      return;
    }
    setCreating(true);
    try {
      await teamApi.createTeam(newName);
      setShowCreate(false);
      setNewName('');
      fetchTeams();
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setCreateError(axiosErr?.response?.data?.error || '创建失败');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (slug: string) => {
    if (!confirm('确定要解散这个团队吗？此操作不可撤销。')) return;
    try {
      await teamApi.deleteTeam(slug);
      fetchTeams();
    } catch {
      alert('操作失败');
    }
  };

  const formatBalance = (cents: number) => {
    return (cents / 100).toFixed(2);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">我的团队</h1>
          <p className="text-gray-500 text-sm mt-1">管理你的 AI 开发团队和 API 额度</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 transition-colors shadow-sm"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          创建团队
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 text-sm rounded-lg px-4 py-3 mb-6">
          {error}
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40" onClick={() => setShowCreate(false)} />
          <div className="relative bg-white rounded-2xl shadow-xl border border-gray-100 w-full max-w-md p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">创建团队</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              {createError && (
                <div className="bg-red-50 border border-red-200 text-red-600 text-sm rounded-lg px-4 py-2.5">
                  {createError}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">团队名称</label>
                <input
                  type="text"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="例如：AI 创业小队"
                />
                <p className="text-xs text-gray-400 mt-1">团队唯一标识将自动生成</p>
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => { setShowCreate(false); setCreateError(''); }}
                  className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                  {creating ? '创建中...' : '确认创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Team list */}
      {loading ? (
        <div className="flex justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      ) : teams.length === 0 ? (
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-12 text-center">
          <svg className="w-16 h-16 mx-auto text-gray-300 mb-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-2">还没有团队</h3>
          <p className="text-gray-500 text-sm mb-6">创建一个团队，开始管理你的 AI API 调用</p>
          <button
            onClick={() => setShowCreate(true)}
            className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 transition-colors shadow-sm"
          >
            创建团队
          </button>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {teams.map((team) => (
            <Link
              key={team.id}
              to={`/dashboard/teams/${team.slug}`}
              className="block bg-white rounded-xl border border-gray-100 shadow-sm hover:shadow-md hover:border-gray-200 transition-all p-5"
            >
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h3 className="font-semibold text-gray-900">{team.name}</h3>
                  <p className="text-xs text-gray-400 mt-0.5">@{team.slug}</p>
                </div>
              </div>
              <div className="flex items-end justify-between">
                <div>
                  <p className="text-xs text-gray-500">余额</p>
                  <p className="text-lg font-bold text-blue-600">
                    ¥{formatBalance(team.balance)}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-xs text-gray-500">成员</p>
                  <p className="text-sm font-medium text-gray-700">
                    {team.members?.length || 0} 人
                  </p>
                </div>
              </div>
              <div className="mt-4 pt-4 border-t border-gray-100 flex justify-between">
                <span
                  className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                    team.status === 'active'
                      ? 'bg-green-50 text-green-600'
                      : 'bg-gray-100 text-gray-500'
                  }`}
                >
                  {team.status === 'active' ? '运行中' : '已禁用'}
                </span>
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleDelete(team.slug);
                  }}
                  className="text-xs text-gray-400 hover:text-red-500 transition-colors"
                >
                  解散
                </button>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
