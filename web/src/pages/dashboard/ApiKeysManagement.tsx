import { useEffect, useState, useCallback } from 'react';
import * as teamApi from '../../api/team';
import type { UserAPIKey } from '../../types/api';

export default function ApiKeysManagement() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [keys, setKeys] = useState<UserAPIKey[]>([]);
  const [busy, setBusy] = useState(false);
  // 已加载完整 key 的缓存 (id → fullKey)
  const [revealedKeys, setRevealedKeys] = useState<Map<number, string>>(new Map());
  const [revealingId, setRevealingId] = useState<number | null>(null);
  // 已复制的 key id（用于显示"已复制"提示）
  const [copiedKeyId, setCopiedKeyId] = useState<number | null>(null);

  // 创建 Key 弹窗
  const [showCreate, setShowCreate] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  // 刚创建的 key（展示一次完整密钥）
  const [freshKey, setFreshKey] = useState<{ id: number; key: string; name: string } | null>(null);

  const fetchKeys = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await teamApi.listMyKeys();
      setKeys(res.data.data || []);
    } catch {
      setError('获取 API Key 列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchKeys(); }, [fetchKeys]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setBusy(true);
      setError('');
      const res = await teamApi.createMyKey(newKeyName.trim() || 'Default');
      const data = res.data.data;
      if (data) {
        setFreshKey({ id: data.id, key: data.key, name: data.name || 'Default' });
        setShowCreate(false);
        setNewKeyName('');
        fetchKeys();
      }
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '创建失败';
      setError(msg);
    } finally {
      setBusy(false);
    }
  };

  const handleToggle = async (k: UserAPIKey) => {
    const isDisabling = k.status === 1;
    const action = isDisabling ? '禁用' : '启用';
    if (!confirm(`确定要${action} API Key "${k.name}"？${isDisabling ? '禁用后该 Key 将立即失效。' : ''}`)) return;
    try {
      setBusy(true);
      setError('');
      await teamApi.toggleMyKey(k.id);
      fetchKeys();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || `${action}失败`;
      setError(msg);
    } finally {
      setBusy(false);
    }
  };

  const handleDelete = async (k: UserAPIKey) => {
    if (!confirm(`确定要删除 API Key "${k.name}"？此操作不可撤销，该 Key 将立即失效。`)) return;
    try {
      setBusy(true);
      setError('');
      await teamApi.deleteMyKey(k.id);
      setRevealedKeys((prev) => { const next = new Map(prev); next.delete(k.id); return next; });
      fetchKeys();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || '删除失败';
      setError(msg);
    } finally {
      setBusy(false);
    }
  };

  const toggleVisible = async (id: number) => {
    if (revealedKeys.has(id)) {
      // 已加载，切换隐藏
      setRevealedKeys((prev) => { const next = new Map(prev); next.delete(id); return next; });
      return;
    }
    // 未加载，请求完整 key
    try {
      setRevealingId(id);
      const res = await teamApi.getMyKey(id);
      const data = res.data.data;
      if (data?.key) {
        setRevealedKeys((prev) => new Map(prev).set(id, data.key));
      }
    } catch {
      // ignore
    } finally {
      setRevealingId(null);
    }
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
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-900">API Key</h1>
          <p className="text-gray-500 text-sm mt-1">管理你的个人 API Key，用于调用 AI 模型</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          disabled={busy}
          className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50 transition-colors shadow-sm"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          创建 Key
        </button>
      </div>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 rounded-lg p-3">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      )}

      {/* 刚创建的 Key 提示 */}
      {freshKey && (
        <div className="mb-4 bg-green-50 border border-green-200 rounded-xl p-5">
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2">
                <svg className="w-5 h-5 text-green-600 shrink-0" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <span className="text-sm font-semibold text-green-800">API Key 创建成功 - {freshKey.name}</span>
              </div>
              <p className="text-xs text-green-600 mb-2">API Key 已创建。你可以随时点击列表中的 <svg className="w-3.5 h-3.5 inline" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg> 图标再次查看完整密钥。</p>
              <div className="bg-white rounded-lg p-3 border border-green-100">
                <div className="flex items-center justify-between">
                  <code className="text-sm font-mono text-gray-900 break-all select-all">{freshKey.key}</code>
                  <button
                    onClick={() => { navigator.clipboard.writeText(freshKey.key); setCopiedKeyId(freshKey.id); setTimeout(() => setCopiedKeyId(null), 2000); }}
                    className="p-1.5 text-gray-400 hover:text-green-600 rounded-md shrink-0 ml-2 transition-colors"
                    title="复制"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
            <button
              onClick={() => setFreshKey(null)}
              className="text-green-400 hover:text-green-600 shrink-0 ml-3"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/40" onClick={() => { setShowCreate(false); setNewKeyName(''); }} />
          <div className="relative bg-white rounded-2xl shadow-xl border border-gray-100 w-full max-w-md p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">创建 API Key</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1.5">Key 名称</label>
                <input
                  type="text"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="例如：Production / Dev / Testing"
                  autoFocus
                />
                <p className="text-xs text-gray-400 mt-1">方便辨识不同用途的 Key</p>
              </div>
              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => { setShowCreate(false); setNewKeyName(''); }}
                  className="flex-1 rounded-lg border border-gray-300 px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={busy}
                  className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                  {busy ? '创建中...' : '确认创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Key List */}
      {keys.length === 0 ? (
        <div className="text-center py-16 bg-white rounded-xl border border-gray-200">
          <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          <h3 className="mt-4 text-sm font-medium text-gray-900">尚未创建 API Key</h3>
          <p className="text-xs text-gray-400 mt-1">创建后即可使用兼容 OpenAI 的接口调用各种 AI 模型</p>
          <button
            onClick={() => setShowCreate(true)}
            disabled={busy}
            className="mt-6 inline-flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50 transition-colors shadow-sm"
          >
            创建 API Key
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {keys.map((k) => (
            <div key={k.id} className="bg-white rounded-xl border border-gray-200 hover:border-gray-300 transition-colors">
              <div className="px-5 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3 min-w-0 flex-1">
                  <div className={`w-9 h-9 rounded-lg flex items-center justify-center shrink-0 ${k.status === 1 ? 'bg-blue-50' : 'bg-gray-100'}`}>
                    <svg className={`w-4.5 h-4.5 ${k.status === 1 ? 'text-blue-600' : 'text-gray-400'}`} fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                    </svg>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-semibold text-gray-900 truncate">{k.name}</span>
                      <span className={`text-xs font-medium px-2 py-0.5 rounded-full shrink-0 ${
                        k.status === 1 ? 'bg-green-50 text-green-600' : 'bg-orange-50 text-orange-500'
                      }`}>
                        {k.status === 1 ? '启用' : '已禁用'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      {revealingId === k.id ? (
                        <span className="text-xs text-gray-400">加载中...</span>
                      ) : (
                        <code className={`text-xs font-mono truncate ${k.status === 2 ? 'text-gray-300 line-through' : revealedKeys.has(k.id) ? 'text-gray-700' : 'text-gray-400'}`}>
                          {revealedKeys.has(k.id) ? revealedKeys.get(k.id) : (k.key_mask || '—')}
                        </code>
                      )}
                      <button
                        onClick={() => toggleVisible(k.id)}
                        disabled={revealingId === k.id}
                        className="text-gray-400 hover:text-blue-600 shrink-0 disabled:opacity-50"
                        title={revealedKeys.has(k.id) ? '隐藏' : '显示完整 Key'}
                      >
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                          {revealedKeys.has(k.id) ? (
                            <path strokeLinecap="round" strokeLinejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                          ) : (
                            <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                          )}
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-1.5 shrink-0 ml-3">
                  {revealedKeys.has(k.id) && (
                    <button
                      onClick={() => { navigator.clipboard.writeText(revealedKeys.get(k.id) || ''); setCopiedKeyId(k.id); setTimeout(() => setCopiedKeyId(null), 2000); }}
                      className="px-2.5 py-1.5 text-xs font-medium text-gray-500 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
                      title="复制完整 Key"
                    >
                      {copiedKeyId === k.id ? '已复制' : '复制'}
                    </button>
                  )}
                  <button
                    onClick={() => handleToggle(k)}
                    disabled={busy}
                    className={`px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors disabled:opacity-50 ${
                      k.status === 2
                        ? 'text-green-600 bg-green-50 hover:bg-green-100'
                        : 'text-gray-500 bg-gray-50 hover:bg-gray-100'
                    }`}
                  >
                    {k.status === 2 ? '启用' : '禁用'}
                  </button>
                  <button
                    onClick={() => handleDelete(k)}
                    disabled={busy}
                    className="px-2.5 py-1.5 text-xs font-medium text-red-500 bg-red-50 rounded-lg hover:bg-red-100 disabled:opacity-50 transition-colors"
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Footer hint */}
      {keys.length > 0 && (
        <p className="text-xs text-gray-400 mt-4 px-1">
          使用以上任意 Key 通过兼容 OpenAI 的接口调用模型，消耗的额度将从你所在团队的余额中扣除。
        </p>
      )}
    </div>
  );
}
