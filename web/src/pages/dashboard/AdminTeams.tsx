import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { AdminTeamItem } from '../../types/api';

function formatBalance(cents: number) {
  return (cents / 100).toFixed(2);
}

function formatCentsToYuan(cents: number) {
  return (cents / 100).toFixed(2);
}

export default function AdminTeams() {
  const [teams, setTeams] = useState<AdminTeamItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Recharge modal
  const [rechargeSlug, setRechargeSlug] = useState<string | null>(null);
  const [rechargeAmount, setRechargeAmount] = useState('');
  const [rechargeRemark, setRechargeRemark] = useState('');
  const [rechargeLoading, setRechargeLoading] = useState(false);
  const [rechargeError, setRechargeError] = useState('');
  const [rechargeSuccess, setRechargeSuccess] = useState('');

  const fetchTeams = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await adminApi.listTeams();
      setTeams(res.data.data || []);
    } catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setError(ae?.response?.data?.error || '获取团队列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchTeams(); }, [fetchTeams]);

  const handleRecharge = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rechargeSlug) return;
    const n = parseFloat(rechargeAmount);
    if (isNaN(n) || n <= 0) {
      setRechargeError('请输入有效金额');
      return;
    }
    const amount = Math.round(n * 100);
    setRechargeError('');
    setRechargeSuccess('');
    setRechargeLoading(true);
    try {
      await adminApi.rechargeTeam(rechargeSlug, amount, rechargeRemark);
      setRechargeSuccess(`成功为 ${rechargeSlug} 充值 ¥${formatCentsToYuan(amount)}`);
      setRechargeAmount('');
      setRechargeRemark('');
      fetchTeams();
    } catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setRechargeError(ae?.response?.data?.error || '充值失败');
    } finally {
      setRechargeLoading(false);
    }
  };

  const openRecharge = (slug: string) => {
    setRechargeSlug(slug);
    setRechargeAmount('');
    setRechargeRemark('');
    setRechargeError('');
    setRechargeSuccess('');
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">团队列表</h2>
        <button
          onClick={fetchTeams}
          className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 transition-colors"
        >
          刷新
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm mb-6">
          {error}
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4">
          <p className="text-xs text-gray-500 mb-1">团队总数</p>
          <p className="text-2xl font-bold text-gray-900">{teams.length}</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4">
          <p className="text-xs text-gray-500 mb-1">总余额</p>
          <p className="text-2xl font-bold text-blue-600">
            ¥{formatBalance(teams.reduce((sum, t) => sum + (t.balance || 0), 0))}
          </p>
        </div>
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4">
          <p className="text-xs text-gray-500 mb-1">总成员</p>
          <p className="text-2xl font-bold text-gray-900">
            {teams.reduce((sum, t) => sum + (t.member_count || 0), 0)}
          </p>
        </div>
      </div>

      {/* Team List */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="border-b border-gray-200">
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">团队</th>
              <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">标识</th>
              <th className="text-center px-5 py-3 text-xs font-medium text-gray-500 uppercase">成员</th>
              <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">余额</th>
              <th className="text-center px-5 py-3 text-xs font-medium text-gray-500 uppercase">状态</th>
              <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {teams.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-12 text-gray-400 text-sm">暂无团队</td>
              </tr>
            ) : (
              teams.map((team) => (
                <tr key={team.id} className="hover:bg-gray-50/50">
                  <td className="px-5 py-4">
                    <span className="text-sm font-medium text-gray-900">{team.name}</span>
                  </td>
                  <td className="px-5 py-4">
                    <span className="text-xs text-gray-500">@{team.slug}</span>
                  </td>
                  <td className="px-5 py-4 text-center">
                    <span className="text-sm text-gray-600">{team.member_count}</span>
                  </td>
                  <td className="px-5 py-4 text-right">
                    <span className="text-sm font-semibold text-blue-600">¥{formatBalance(team.balance)}</span>
                  </td>
                  <td className="px-5 py-4 text-center">
                    <span className={`inline-flex text-xs font-medium px-2 py-0.5 rounded ${
                      team.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
                    }`}>
                      {team.status === 'active' ? '运行中' : '已禁用'}
                    </span>
                  </td>
                  <td className="px-5 py-4 text-right">
                    <button
                      onClick={() => openRecharge(team.slug)}
                      className="text-xs font-medium text-blue-600 hover:text-blue-700 transition-colors"
                    >
                      充值
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Recharge Modal */}
      {rechargeSlug && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40" onClick={() => setRechargeSlug(null)} />
          <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">团队充值 - {rechargeSlug}</h3>
              <button
                onClick={() => setRechargeSlug(null)}
                className="text-gray-400 hover:text-gray-600"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <form onSubmit={handleRecharge} className="space-y-4">
              {rechargeSuccess && (
                <div className="bg-green-50 border border-green-200 text-green-700 rounded-lg px-4 py-2.5 text-sm">
                  {rechargeSuccess}
                </div>
              )}
              {rechargeError && (
                <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm">
                  {rechargeError}
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">充值金额 (元)</label>
                <input
                  type="number"
                  value={rechargeAmount}
                  onChange={(e) => setRechargeAmount(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="输入金额"
                  min="0.01"
                  step="0.01"
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">备注 <span className="text-gray-400 font-normal">(可选)</span></label>
                <input
                  type="text"
                  value={rechargeRemark}
                  onChange={(e) => setRechargeRemark(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="充值原因或备注"
                  maxLength={200}
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setRechargeSlug(null)}
                  className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={rechargeLoading}
                  className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {rechargeLoading ? '处理中...' : '确认充值'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
