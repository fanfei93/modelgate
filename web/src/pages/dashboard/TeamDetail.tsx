import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import * as teamApi from '../../api/team';
import { useAuth } from '../../hooks/useAuth';
import type { Team, QuotaInfo, LogItem } from '../../types/api';

function formatBalance(cents: number) {
  return (cents / 100).toFixed(2);
}

function formatTime(ts: number) {
  const d = new Date(ts * 1000);
  return d.toLocaleString('zh-CN');
}

// ============ 所有者视图 ============

function OwnerView({ team, slug, user, fetchTeam }: {
  team: Team;
  slug: string;
  user: { id: number };
  fetchTeam: () => void;
}) {
  const [addEmailsText, setAddEmailsText] = useState('');
  const [adding, setAdding] = useState(false);
  const [memberError, setMemberError] = useState('');
  const [showAddModal, setShowAddModal] = useState(false);
  const [addResult, setAddResult] = useState<{ added: number; failed: string[] } | null>(null);

  const [quotaMember, setQuotaMember] = useState<{ id: number; name: string } | null>(null);
  const [quotaAmount, setQuotaAmount] = useState('');
  const [quotaLoading, setQuotaLoading] = useState(false);
  const [quotaError, setQuotaError] = useState('');
  const [quotaInfo, setQuotaInfo] = useState<QuotaInfo | null>(null);
  const [quotaInfoLoading, setQuotaInfoLoading] = useState(false);

  const totalAllocated = team.members?.reduce((sum, m) => sum + (m.quota_allocated || 0), 0) || 0;

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    const lines = addEmailsText.split(/\n/).map((l) => l.trim()).filter((l) => l);
    const entries = lines.map((line) => {
      const parts = line.split(/\s+/);
      const email = parts[parts.length - 1];
      const name = parts.length > 1 ? parts.slice(0, -1).join(' ') : '';
      return { name, email };
    }).filter((e) => e.email);
    if (entries.length === 0) { setMemberError('请输入至少一条记录'); return; }
    setMemberError('');
    setAdding(true);
    try {
      const res = await teamApi.addMembers(slug, entries);
      setAddResult(res.data.data || { added: 0, failed: [] });
      setAddEmailsText('');
      fetchTeam();
    } catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setMemberError(ae?.response?.data?.error || '添加失败');
    } finally { setAdding(false); }
  };

  const handleRemoveMember = async (memberId: number) => {
    if (!confirm('确定移除该成员？')) return;
    try { await teamApi.removeMember(slug, memberId); fetchTeam(); }
    catch { alert('操作失败'); }
  };

  const handleCancelInvitation = async (invitationId: number) => {
    if (!confirm('确定取消该邀请？')) return;
    try { await teamApi.cancelInvitation(slug, invitationId); fetchTeam(); }
    catch { alert('操作失败'); }
  };

  const handleOpenQuota = async (memberId: number, memberName: string) => {
    setQuotaMember({ id: memberId, name: memberName });
    setQuotaAmount('');
    setQuotaError('');
    setQuotaInfoLoading(true);
    setQuotaInfo(null);
    try {
      const res = await teamApi.getMemberQuota(slug, memberId);
      const info = res.data.data;
      if (info) {
        setQuotaInfo(info);
        if (info.quota_allocated > 0) setQuotaAmount((info.quota_allocated / 100).toFixed(2));
      }
    } catch { /* ignore */ }
    finally { setQuotaInfoLoading(false); }
  };

  const handleSetQuota = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!quotaMember) return;
    const n = parseFloat(quotaAmount);
    if (isNaN(n) || n <= 0) { setQuotaError('请输入有效金额'); return; }
    const amount = Math.round(n * 100);
    setQuotaError('');
    setQuotaLoading(true);
    try {
      await teamApi.setMemberQuota(slug, quotaMember.id, amount);
      setQuotaMember(null); fetchTeam();
    } catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setQuotaError(ae?.response?.data?.error || '操作失败');
    } finally { setQuotaLoading(false); }
  };

  const handleRevokeQuota = async () => {
    if (!quotaMember || !confirm('确定回收该成员的所有额度？')) return;
    setQuotaLoading(true);
    try { await teamApi.revokeMemberQuota(slug, quotaMember.id); setQuotaMember(null); fetchTeam(); }
    catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setQuotaError(ae?.response?.data?.error || '操作失败');
    } finally { setQuotaLoading(false); }
  };

  return (
    <div>
      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
        <StatCard label="团队总额度" value={formatBalance(team.balance)} color="text-gray-900" />
        <StatCard label="已分配额度" value={formatBalance(totalAllocated)} color="text-blue-600" />
        <StatCard label="剩余可分配" value={formatBalance(Math.max(0, team.balance - totalAllocated))} color={team.balance - totalAllocated >= 0 ? 'text-green-600' : 'text-red-500'} />
        <StatCard label="成员数" value={`${team.members?.length || 0}`} color="text-gray-900" suffix="人" />
        <StatCard label="已创建 Key" value={`${team.members?.filter((m) => m.api_key_mask).length || 0}`} color="text-gray-900" suffix="个" />
      </div>

      {/* 成员管理 */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-bold text-gray-900">成员管理</h3>
          <button
            onClick={() => { setShowAddModal(true); setMemberError(''); }}
            className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            添加成员
          </button>
        </div>

        <div className="border border-gray-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50">
              <tr className="border-b border-gray-200">
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">昵称</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">邮箱</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">角色</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">额度</th>
                <th className="text-left px-5 py-3 text-xs font-medium text-gray-500 uppercase">API Key</th>
                <th className="text-right px-5 py-3 text-xs font-medium text-gray-500 uppercase">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {team.members?.map((m) => (
                <tr key={m.id} className="hover:bg-gray-50/50">
                  <td className="px-5 py-4">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-900">{m.user?.username || '-'}</span>
                      {m.role === 'owner' && <span className="text-xs text-purple-600 bg-purple-50 px-1.5 py-0.5 rounded">所有者</span>}
                    </div>
                  </td>
                  <td className="px-5 py-4 text-gray-500">{m.user?.email || '-'}</td>
                  <td className="px-5 py-4">
                    <span className={`inline-flex text-xs font-medium px-2 py-0.5 rounded ${m.role === 'owner' ? 'bg-purple-100 text-purple-700' : 'bg-gray-100 text-gray-600'}`}>
                      {m.role === 'owner' ? '拥有者' : '成员'}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    {m.quota_allocated > 0 ? (
                      <div className="max-w-[160px]">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs font-medium text-gray-900">¥{formatBalance(m.quota_allocated)}</span>
                          <span className="text-[11px] text-gray-500">{m.quota_used > 0 ? Math.min(100, Math.round((m.quota_used / m.quota_allocated) * 100)) : 0}%</span>
                        </div>
                        <div className="w-full h-1.5 bg-gray-100 rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${
                              (m.quota_used / m.quota_allocated) > 0.8 ? 'bg-red-500' :
                              (m.quota_used / m.quota_allocated) > 0.5 ? 'bg-amber-500' :
                              'bg-emerald-500'
                            }`}
                            style={{ width: `${Math.min(100, Math.round((m.quota_used / m.quota_allocated) * 100))}%` }}
                          />
                        </div>
                        <p className="text-[11px] text-gray-400 mt-0.5">已用 ¥{formatBalance(m.quota_used)}</p>
                      </div>
                    ) : <span className="text-xs text-gray-400">未设置</span>}
                  </td>
                  <td className="px-5 py-4">
                    {m.api_key_mask ? <span className="text-xs font-mono text-green-600">{m.api_key_mask}</span> : <span className="text-xs text-gray-400">未创建</span>}
                  </td>
                  <td className="px-5 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      {m.role !== 'owner' && (
                        <>
                          <button onClick={() => handleOpenQuota(m.id, m.user?.username || `#${m.id}`)}
                            className="text-xs text-blue-600 hover:text-blue-800 font-medium transition-colors">设置额度</button>
                          <button onClick={() => handleRemoveMember(m.id)}
                            className="p-1.5 text-gray-400 hover:text-red-500 transition-colors" title="移除成员">
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
                            </svg>
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* 待处理邀请 */}
        {team.invitations && team.invitations.filter(inv => inv.status === 'pending').length > 0 && (
          <div className="mt-6">
            <h4 className="text-sm font-semibold text-gray-700 mb-3">待接受邀请</h4>
            <div className="border border-yellow-200 rounded-lg overflow-hidden">
              <table className="w-full text-sm">
                <thead className="bg-yellow-50">
                  <tr className="border-b border-yellow-200">
                    <th className="text-left px-5 py-2.5 text-xs font-medium text-yellow-700 uppercase">昵称</th>
                    <th className="text-left px-5 py-2.5 text-xs font-medium text-yellow-700 uppercase">邮箱</th>
                    <th className="text-left px-5 py-2.5 text-xs font-medium text-yellow-700 uppercase">状态</th>
                    <th className="text-right px-5 py-2.5 text-xs font-medium text-yellow-700 uppercase">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-yellow-100">
                  {team.invitations.filter(inv => inv.status === 'pending').map((inv) => (
                    <tr key={inv.id} className="hover:bg-yellow-50/30">
                      <td className="px-5 py-3"><span className="text-sm text-gray-700">{inv.name || '-'}</span></td>
                      <td className="px-5 py-3 text-gray-500">{inv.email}</td>
                      <td className="px-5 py-3"><span className="inline-flex text-xs font-medium px-2 py-0.5 rounded bg-yellow-100 text-yellow-700">待注册</span></td>
                      <td className="px-5 py-3 text-right">
                        <button onClick={() => handleCancelInvitation(inv.id)} className="text-xs text-gray-400 hover:text-red-500 transition-colors">取消邀请</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* 额度设置弹窗 */}
        {quotaMember && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-black/40" onClick={() => setQuotaMember(null)} />
            <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md p-6">
              <div className="flex items-center justify-between mb-4">
                <h4 className="text-lg font-semibold text-gray-900">设置额度 - {quotaMember.name}</h4>
                <button onClick={() => { setQuotaMember(null); setQuotaError(''); }} className="text-gray-400 hover:text-gray-600">
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              {quotaInfoLoading ? (
                <div className="flex justify-center py-8"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" /></div>
              ) : (
                <form onSubmit={handleSetQuota} className="space-y-4">
                  {quotaInfo && (
                    <div className="bg-gray-50 rounded-lg p-3 space-y-1 text-sm">
                      {quotaInfo.quota_allocated > 0 ? (
                        <>
                          <p className="text-gray-700">已分配: <span className="font-semibold">¥{formatBalance(quotaInfo.quota_allocated)}</span></p>
                          <p className="text-gray-700">已使用: <span className="font-semibold text-orange-500">¥{formatBalance(quotaInfo.quota_used)}</span></p>
                          <p className="text-gray-700">剩余: <span className="font-semibold text-green-600">¥{formatBalance(quotaInfo.quota_remain)}</span></p>
                        </>
                      ) : <p className="text-gray-500">尚未设置额度</p>}
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1.5">分配金额 (元)</label>
                    <input type="number" value={quotaAmount} onChange={(e) => setQuotaAmount(e.target.value)}
                      className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                      placeholder="输入金额" min="0.01" step="0.01" />
                  </div>
                  {quotaError && <div className="bg-red-50 border border-red-200 text-red-600 text-sm rounded-lg px-4 py-2.5">{quotaError}</div>}
                  <div className="flex gap-3 pt-2">
                    {quotaInfo && quotaInfo.quota_allocated > 0 && (
                      <button type="button" onClick={handleRevokeQuota} disabled={quotaLoading}
                        className="rounded-lg border border-red-300 px-4 py-2.5 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50">回收额度</button>
                    )}
                    <button type="button" onClick={() => { setQuotaMember(null); setQuotaError(''); }}
                      className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
                    <button type="submit" disabled={quotaLoading}
                      className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50">
                      {quotaLoading ? '处理中...' : '确认设置'}
                    </button>
                  </div>
                </form>
              )}
            </div>
          </div>
        )}

        {/* 添加成员弹窗 */}
        {showAddModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-black/40" onClick={() => setShowAddModal(false)} />
            <div className="relative bg-white rounded-xl shadow-xl w-full max-w-lg p-6">
              <div className="flex items-center justify-between mb-4">
                <h4 className="text-lg font-semibold text-gray-900">批量添加成员</h4>
                <button onClick={() => { setShowAddModal(false); setMemberError(''); setAddResult(null); }} className="text-gray-400 hover:text-gray-600">
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              {!addResult ? (
                <form onSubmit={handleAddMember} className="space-y-4">
                  <p className="text-sm text-gray-600">在下方粘贴成员信息。每行一条，格式为"昵称 邮箱"。</p>
                  <textarea value={addEmailsText} onChange={(e) => setAddEmailsText(e.target.value)} rows={8}
                    className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                    placeholder={`张三 zhangsan@example.com\n李四 lisi@example.com\n王五 wangwu@example.com`} />
                  {memberError && <p className="text-xs text-red-500">{memberError}</p>}
                  <div className="flex gap-3 pt-2">
                    <button type="button" onClick={() => { setShowAddModal(false); setMemberError(''); setAddEmailsText(''); }}
                      className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
                    <button type="submit" disabled={adding || !addEmailsText.trim()}
                      className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50">
                      {adding ? '处理中...' : `下一步 (${addEmailsText.split(/\n/).filter(l => l.trim()).length})`}
                    </button>
                  </div>
                </form>
              ) : (
                <div className="space-y-4">
                  <p className="text-sm text-gray-700">成功添加 <span className="font-semibold text-green-600">{addResult.added}</span> 人</p>
                  <p className="text-xs text-gray-500">未注册用户将收到邀请，使用该邮箱注册后自动加入团队。</p>
                  {addResult.failed.length > 0 && (
                    <div>
                      <p className="text-sm text-red-600 mb-1">以下成员添加失败：</p>
                      <div className="max-h-40 overflow-y-auto bg-red-50 rounded-lg p-3">
                        {addResult.failed.map((msg, i) => (<p key={i} className="text-xs text-red-700">{msg}</p>))}
                      </div>
                    </div>
                  )}
                  <button type="button" onClick={() => { setAddResult(null); setAddEmailsText(''); }}
                    className="w-full rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700">继续添加</button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============ 成员视图 ============

function MemberView({ team, slug }: { team: Team; slug: string }) {
  const { user } = useAuth();
  const [quotaInfo, setQuotaInfo] = useState<QuotaInfo | null>(null);
  const [quotaLoading, setQuotaLoading] = useState(true);
  const [logs, setLogs] = useState<LogItem[]>([]);
  const [logsLoading, setLogsLoading] = useState(true);
  const [logsError, setLogsError] = useState('');

  // 找到当前用户对应的成员记录
  const myMember = team.members?.find((m) => m.user_id === user?.id);

  const fetchQuota = useCallback(async () => {
    if (!myMember || !slug) return;
    setQuotaLoading(true);
    try {
      const res = await teamApi.getMemberQuota(slug, myMember.id);
      setQuotaInfo(res.data.data || null);
    } catch { /* ignore */ }
    finally { setQuotaLoading(false); }
  }, [slug, myMember?.id]);

  const fetchLogs = useCallback(async () => {
    if (!slug) return;
    setLogsLoading(true);
    setLogsError('');
    try {
      const res = await teamApi.getMemberLogs(slug);
      setLogs(res.data.data || []);
    } catch (err: unknown) {
      const ae = err as { response?: { data?: { error?: string } } };
      setLogsError(ae?.response?.data?.error || '获取日志失败');
    } finally { setLogsLoading(false); }
  }, [slug]);

  useEffect(() => { fetchQuota(); fetchLogs(); }, [fetchQuota, fetchLogs]);

  const allocated = quotaInfo?.quota_allocated || myMember?.quota_allocated || 0;
  const used = quotaInfo?.quota_used || myMember?.quota_used || 0;
  const remain = quotaInfo?.quota_remain || 0;
  return (
    <div>
      {/* 我的额度 */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6 mb-6">
        <h3 className="text-lg font-bold text-gray-900 mb-4">我的额度</h3>
        {quotaLoading ? (
          <div className="flex justify-center py-4"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" /></div>
        ) : (
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-gray-50 rounded-lg p-4 text-center">
              <p className="text-xs text-gray-500 mb-1">已分配</p>
              <p className="text-xl font-bold text-gray-900">¥{formatBalance(allocated)}</p>
            </div>
            <div className="bg-orange-50 rounded-lg p-4 text-center">
              <p className="text-xs text-gray-500 mb-1">已使用</p>
              <p className="text-xl font-bold text-orange-600">¥{formatBalance(used)}</p>
            </div>
            <div className="bg-green-50 rounded-lg p-4 text-center">
              <p className="text-xs text-gray-500 mb-1">剩余</p>
              <p className="text-xl font-bold text-green-600">¥{formatBalance(remain)}</p>
            </div>
          </div>
        )}
      </div>

      {/* 使用日志 */}
      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-bold text-gray-900">使用日志</h3>
          <button onClick={fetchLogs} disabled={logsLoading}
            className="text-xs text-blue-600 hover:text-blue-800 font-medium disabled:opacity-50">
            刷新
          </button>
        </div>

        {logsLoading ? (
          <div className="flex justify-center py-8"><div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600" /></div>
        ) : logsError ? (
          <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm">{logsError}</div>
        ) : logs.length === 0 ? (
          <div className="text-center py-8 text-gray-400 text-sm">暂无使用记录</div>
        ) : (
          <div className="border border-gray-200 rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50">
                <tr className="border-b border-gray-200">
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-gray-500">时间</th>
                  <th className="text-left px-4 py-2.5 text-xs font-medium text-gray-500">模型</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-gray-500">Token 消耗</th>
                  <th className="text-right px-4 py-2.5 text-xs font-medium text-gray-500">费用</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-gray-50/50">
                    <td className="px-4 py-3 text-gray-600">{formatTime(log.created_at)}</td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-gray-900">{log.model_name || '-'}</span>
                    </td>
                    <td className="px-4 py-3 text-right text-gray-600">
                      <span className="text-xs mr-1">P:</span>{log.prompt_tokens}
                      <span className="text-xs mx-1">C:</span>{log.completion_tokens}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span className="text-sm font-medium text-gray-900">{formatBalance(log.quota)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

// ============ 辅助组件 ============

function StatCard({ label, value, color, suffix }: { label: string; value: string; color: string; suffix?: string }) {
  return (
    <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4">
      <p className="text-xs text-gray-500 mb-1">{label}</p>
      <p className={`text-2xl font-bold ${color}`}>{value}</p>
      {suffix && <p className="text-xs text-gray-400 mt-0.5">{suffix}</p>}
    </div>
  );
}

// ============ 主组件 ============

export default function TeamDetail() {
  const { slug } = useParams<{ slug: string }>();
  const { user } = useAuth();
  const [team, setTeam] = useState<Team | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const fetchTeam = useCallback(async () => {
    if (!slug) return;
    try {
      setLoading(true);
      const res = await teamApi.getTeam(slug);
      setTeam(res.data.data || null);
    } catch {
      setError('获取团队信息失败');
    } finally { setLoading(false); }
  }, [slug]);

  useEffect(() => { fetchTeam(); }, [fetchTeam]);

  if (loading) {
    return <div className="flex justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>;
  }
  if (error || !team) {
    return <div className="bg-red-50 border border-red-200 text-red-600 rounded-xl px-4 py-3 text-sm">{error || '团队不存在'}</div>;
  }

  const isOwner = user?.id === team.owner_id;

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{team.name}</h1>
        <p className="text-gray-500 text-sm mt-1">@{team.slug}</p>
      </div>

      {isOwner ? (
        <OwnerView team={team} slug={slug!} user={user!} fetchTeam={fetchTeam} />
      ) : (
        <MemberView team={team} slug={slug!} />
      )}
    </div>
  );
}
