import { useEffect, useState, useCallback } from 'react';
import * as teamApi from '../../api/team';
import type { APIKeySummary } from '../../types/api';

export default function ApiKeysManagement() {
  const [keys, setKeys] = useState<APIKeySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [creatingSlug, setCreatingSlug] = useState<string | null>(null);
  const [shownKeys, setShownKeys] = useState<Record<string, string>>({});
  const [copiedSlug, setCopiedSlug] = useState<string | null>(null);

  const fetchKeys = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await teamApi.getMyApiKeys();
      setKeys(res.data.data || []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '加载 API Keys 失败';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  const handleCreateKey = async (slug: string) => {
    try {
      setCreatingSlug(slug);
      const res = await teamApi.createMyKey(slug);
      const key = res.data.data?.key || '';
      setShownKeys((prev) => ({ ...prev, [slug]: key }));
      await fetchKeys();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '创建 API Key 失败';
      setError(msg);
    } finally {
      setCreatingSlug(null);
    }
  };

  const handleShowKey = async (slug: string) => {
    try {
      setCreatingSlug(slug);
      const res = await teamApi.getMyKey(slug);
      const key = res.data.data?.key || '';
      setShownKeys((prev) => ({ ...prev, [slug]: key }));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '获取 API Key 失败';
      setError(msg);
    } finally {
      setCreatingSlug(null);
    }
  };

  const hideKey = (slug: string) => {
    setShownKeys((prev) => {
      const next = { ...prev };
      delete next[slug];
      return next;
    });
  };

  const handleToggleKey = async (slug: string) => {
    const item = keys.find((k) => k.team_slug === slug);
    const isDisabling = item?.key_status === 1;
    const action = isDisabling ? '禁用' : '启用';
    if (!confirm(`确定要${action}此 API Key？${isDisabling ? '禁用后该 Key 将立即失效。' : ''}`)) return;
    try {
      setCreatingSlug(slug);
      setError('');
      await teamApi.toggleMyKey(slug);
      await fetchKeys();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `${action} API Key 失败`;
      setError(msg);
    } finally {
      setCreatingSlug(null);
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
      <h1 className="text-xl font-bold text-gray-900 mb-6">API Keys 管理</h1>

      {error && (
        <div className="mb-4 bg-red-50 border border-red-200 rounded-lg p-3">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      )}

      {keys.length === 0 ? (
        <div className="text-center py-16 bg-white rounded-xl border border-gray-200">
          <svg className="mx-auto h-12 w-12 text-gray-300" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
          </svg>
          <p className="mt-4 text-sm text-gray-500">你还没有加入任何团队</p>
          <p className="text-xs text-gray-400 mt-1">加入团队后，可以在这里管理你的 API Key</p>
        </div>
      ) : (
        <div className="space-y-4">
          <p className="text-sm text-gray-500">
            以下是你加入的所有团队及其 API Key 状态。每个团队拥有独立的 Key，调用额度从对应团队余额中扣除。
          </p>
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100 bg-gray-50">
                  <th className="text-left text-xs font-medium text-gray-500 uppercase px-5 py-3">
                    团队
                  </th>
                  <th className="text-left text-xs font-medium text-gray-500 uppercase px-5 py-3">
                    角色
                  </th>
                  <th className="text-left text-xs font-medium text-gray-500 uppercase px-5 py-3">
                    API Key
                  </th>
                  <th className="text-right text-xs font-medium text-gray-500 uppercase px-5 py-3">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {keys.map((item) => (
                  <tr key={item.team_id} className="hover:bg-gray-50/50">
                    <td className="px-5 py-4">
                      <span className="text-sm font-medium text-gray-900">{item.team_name}</span>
                      <span className="text-xs text-gray-400 ml-2">{item.team_slug}</span>
                    </td>
                    <td className="px-5 py-4">
                      <span
                        className={`inline-flex text-xs font-medium px-2 py-0.5 rounded ${
                          item.role === 'owner'
                            ? 'bg-purple-100 text-purple-700'
                            : 'bg-gray-100 text-gray-600'
                        }`}
                      >
                        {item.role === 'owner' ? '拥有者' : '成员'}
                      </span>
                    </td>
                    <td className="px-5 py-4">
                      {item.has_key ? (
                        shownKeys[item.team_slug] ? (
                          <div className="flex items-center gap-1.5">
                            <code className="text-xs font-mono text-gray-900 break-all">
                              {shownKeys[item.team_slug]}
                            </code>
                            <button
                              onClick={() => { navigator.clipboard.writeText(shownKeys[item.team_slug]); setCopiedSlug(item.team_slug); setTimeout(() => setCopiedSlug(null), 2000); }}
                              className="shrink-0 p-0.5 text-gray-400 hover:text-blue-600"
                              title="复制"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                              </svg>
                            </button>
                            <button
                              onClick={() => hideKey(item.team_slug)}
                              className="shrink-0 p-0.5 text-gray-400 hover:text-gray-600"
                              title="隐藏"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <span className={`text-xs font-mono ${item.key_status === 2 ? 'text-gray-300 line-through' : 'text-gray-500'}`}>
                              {item.api_key_mask}
                            </span>
                            {item.key_status === 2 && (
                              <span className="text-xs text-orange-500 bg-orange-50 px-1.5 py-0.5 rounded font-medium">已禁用</span>
                            )}
                            <button
                              onClick={() => handleShowKey(item.team_slug)}
                              disabled={creatingSlug === item.team_slug}
                              className="shrink-0 p-0.5 text-gray-400 hover:text-blue-600 disabled:text-gray-300"
                              title="显示完整 Key"
                            >
                              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                              </svg>
                            </button>
                          </div>
                        )
                      ) : (
                        <span className="text-xs text-gray-400">未创建</span>
                      )}
                    </td>
                    <td className="px-5 py-4 text-right">
                      {item.has_key ? (
                        <div className="flex items-center justify-end gap-3">
                          <button
                            onClick={() => handleToggleKey(item.team_slug)}
                            disabled={creatingSlug === item.team_slug}
                            className={`text-xs hover:text-red-500 disabled:text-gray-300 ${
                              item.key_status === 2 ? 'text-green-600 hover:text-green-700' : 'text-gray-400'
                            }`}
                          >
                            {creatingSlug === item.team_slug ? '处理中...' : item.key_status === 2 ? '启用' : '禁用'}
                          </button>
                          <button
                            onClick={() => handleCreateKey(item.team_slug)}
                            disabled={creatingSlug === item.team_slug}
                            className="text-xs font-medium text-blue-600 hover:text-blue-700 disabled:text-gray-400"
                          >
                            {creatingSlug === item.team_slug ? '创建中...' : '重新生成'}
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => handleCreateKey(item.team_slug)}
                          disabled={creatingSlug === item.team_slug}
                          className="text-xs font-medium text-blue-600 hover:text-blue-700 disabled:text-gray-400"
                        >
                          {creatingSlug === item.team_slug ? '创建中...' : '创建 Key'}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
