import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { AdminTeamItem, SiteSetting, RechargeLog } from '../../types/api';

function formatBalance(cents: number) {
  return (cents / 100).toFixed(2);
}

function formatCentsToYuan(cents: number) {
  return (cents / 100).toFixed(2);
}

export default function AdminDashboard() {
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

  // Site settings
  const [settings, setSettings] = useState<SiteSetting[]>([]);
  const [settingsLoading, setSettingsLoading] = useState(true);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState('');
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [settingsMsg, setSettingsMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchSettings = useCallback(async () => {
    try {
      setSettingsLoading(true);
      const res = await adminApi.listSettings();
      setSettings(res.data.data || []);
    } catch { /* ignore */ }
    finally { setSettingsLoading(false); }
  }, []);

  useEffect(() => { fetchSettings(); }, [fetchSettings]);

  const handleSaveSetting = async (key: string) => {
    setSettingsSaving(true);
    setSettingsMsg(null);
    try {
      await adminApi.updateSetting(key, editingValue);
      setSettingsMsg({ type: 'success', text: '配置已保存' });
      setEditingKey(null);
      fetchSettings();
    } catch {
      setSettingsMsg({ type: 'error', text: '保存失败' });
    } finally {
      setSettingsSaving(false);
    }
  };

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
      fetchRechargeLogs();
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

  // Recharge logs
  const [rechargeLogs, setRechargeLogs] = useState<RechargeLog[]>([]);
  const [rechargeLogsTotal, setRechargeLogsTotal] = useState(0);
  const [rechargeLogsPage, setRechargeLogsPage] = useState(1);
  const [rechargeLogsLoading, setRechargeLogsLoading] = useState(false);

  const fetchRechargeLogs = useCallback(async (page = 1) => {
    try {
      setRechargeLogsLoading(true);
      const res = await adminApi.listRechargeLogs(page, 10);
      setRechargeLogs(res.data.data || []);
      setRechargeLogsTotal(res.data.total || 0);
      setRechargeLogsPage(page);
    } catch { /* ignore */ }
    finally { setRechargeLogsLoading(false); }
  }, []);

  useEffect(() => { fetchRechargeLogs(); }, [fetchRechargeLogs]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">管理后台</h1>
          <p className="text-gray-500 text-sm mt-1">团队管理与充值</p>
        </div>
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

      {/* Recharge Logs */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden mb-6">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <h2 className="text-base font-bold text-gray-900">充值日志</h2>
          <button
            onClick={() => fetchRechargeLogs(rechargeLogsPage)}
            className="text-xs font-medium text-blue-600 hover:text-blue-700 transition-colors"
          >
            刷新
          </button>
        </div>
        {rechargeLogsLoading ? (
          <div className="flex justify-center py-8"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" /></div>
        ) : (
          <>
            <table className="w-full text-sm">
              <thead className="bg-gray-50">
                <tr className="border-b border-gray-200">
                  <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">时间</th>
                  <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">团队</th>
                  <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">操作人</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">充值金额</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">充值前余额</th>
                  <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">充值后余额</th>
                  <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">IP</th>
                  <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">备注</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {rechargeLogs.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="text-center py-12 text-gray-400 text-sm">暂无充值记录</td>
                  </tr>
                ) : (
                  rechargeLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50/50">
                      <td className="px-5 py-3 text-xs text-gray-500 whitespace-nowrap">{log.created_at}</td>
                      <td className="px-5 py-3 text-sm text-gray-900">{log.team_name}</td>
                      <td className="px-5 py-3 text-sm text-gray-600">{log.operator_name}</td>
                      <td className="px-5 py-3 text-right text-sm font-semibold text-green-600">+¥{formatBalance(log.amount)}</td>
                      <td className="px-5 py-3 text-right text-sm text-gray-500">¥{formatBalance(log.balance_before)}</td>
                      <td className="px-5 py-3 text-right text-sm font-medium text-blue-600">¥{formatBalance(log.balance_after)}</td>
                      <td className="px-5 py-3 text-xs text-gray-400">{log.ip}</td>
                      <td className="px-5 py-3 text-xs text-gray-500 max-w-[160px] truncate">{log.remark || '-'}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
            {rechargeLogsTotal > 10 && (
              <div className="flex items-center justify-between px-5 py-3 border-t border-gray-100">
                <span className="text-xs text-gray-500">
                  共 {rechargeLogsTotal} 条记录，第 {rechargeLogsPage} 页
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => fetchRechargeLogs(rechargeLogsPage - 1)}
                    disabled={rechargeLogsPage <= 1}
                    className="rounded-lg border border-gray-300 px-3 py-1 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40"
                  >
                    上一页
                  </button>
                  <button
                    onClick={() => fetchRechargeLogs(rechargeLogsPage + 1)}
                    disabled={rechargeLogsPage * 10 >= rechargeLogsTotal}
                    className="rounded-lg border border-gray-300 px-3 py-1 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Site Settings */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6 mb-6">
        <h2 className="text-lg font-bold text-gray-900 mb-4">站点配置</h2>
        {settingsMsg && (
          <div className={`mb-4 rounded-lg px-4 py-2.5 text-sm ${settingsMsg.type === 'success' ? 'bg-green-50 border border-green-200 text-green-700' : 'bg-red-50 border border-red-200 text-red-600'}`}>
            {settingsMsg.text}
          </div>
        )}
        {settingsLoading ? (
          <div className="flex justify-center py-8"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" /></div>
        ) : (
          <div className="space-y-3">
            {settings.map((s) => (
              <div key={s.key} className="flex items-center justify-between px-4 py-3 rounded-lg bg-gray-50">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900">{s.comment || s.key}</p>
                  <p className="text-xs text-gray-400 mt-0.5">键名: {s.key}</p>
                </div>
                <div className="flex items-center gap-3 ml-4">
                  {editingKey === s.key ? (
                    <>
                      {s.key === 'menu_arena_visible' || s.key === 'menu_docs_visible' ? (
                        <select
                          value={editingValue}
                          onChange={(e) => setEditingValue(e.target.value)}
                          className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          <option value="true">显示</option>
                          <option value="false">隐藏</option>
                        </select>
                      ) : (
                        <input
                          type="text"
                          value={editingValue}
                          onChange={(e) => setEditingValue(e.target.value)}
                          className="w-64 rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          autoFocus
                        />
                      )}
                      <button
                        onClick={() => handleSaveSetting(s.key)}
                        disabled={settingsSaving}
                        className="rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
                      >
                        保存
                      </button>
                      <button
                        onClick={() => setEditingKey(null)}
                        className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50"
                      >
                        取消
                      </button>
                    </>
                  ) : (
                    <>
                      <span className="text-sm text-gray-700 max-w-[200px] truncate">
                        {s.key === 'menu_arena_visible' || s.key === 'menu_docs_visible'
                          ? (s.value === 'true' ? '显示' : '隐藏')
                          : s.value}
                      </span>
                      <button
                        onClick={() => { setEditingKey(s.key); setEditingValue(s.value); }}
                        className="text-xs font-medium text-blue-600 hover:text-blue-700 transition-colors"
                      >
                        编辑
                      </button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
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
