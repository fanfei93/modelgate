import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { LoginLog } from '../../types/api';

export default function AdminLoginLogs() {
  const [logs, setLogs] = useState<LoginLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 20;

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await adminApi.listLoginLogs(page, pageSize);
      setLogs(res.data.data || []);
      setTotal(res.data.total || 0);
    } catch {
      setError('获取登录日志失败');
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">登录日志</h2>
        <button
          onClick={fetchLogs}
          className="text-sm text-purple-600 hover:text-purple-700 font-medium"
        >
          刷新
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-2.5 text-sm mb-6">
          {error}
        </div>
      )}

      {loading ? (
        <div className="text-center py-12 text-gray-400">加载中...</div>
      ) : logs.length === 0 ? (
        <div className="text-center py-12 text-gray-400">暂无登录日志</div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-3 font-medium text-gray-500">时间</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">用户</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">IP</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">状态</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">原因</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">User-Agent</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3 px-3 text-gray-600 whitespace-nowrap">{log.created_at}</td>
                    <td className="py-3 px-3">
                      <span className="font-medium text-gray-900">{log.username}</span>
                      {log.user_id > 0 && (
                        <span className="text-gray-400 ml-1 text-xs">#{log.user_id}</span>
                      )}
                    </td>
                    <td className="py-3 px-3 text-gray-600 font-mono text-xs">{log.ip}</td>
                    <td className="py-3 px-3">
                      {log.success ? (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700">
                          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                          </svg>
                          成功
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700">
                          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
                          </svg>
                          失败
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-3 text-gray-500 text-xs max-w-[200px] truncate" title={log.reason}>
                      {log.reason || '-'}
                    </td>
                    <td className="py-3 px-3 text-gray-400 text-xs max-w-[240px] truncate" title={log.user_agent}>
                      {log.user_agent || '-'}
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
    </div>
  );
}
