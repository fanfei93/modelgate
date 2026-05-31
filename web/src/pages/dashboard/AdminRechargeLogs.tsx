import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { RechargeLog } from '../../types/api';

function formatBalance(cents: number) {
  return (cents / 100).toFixed(2);
}

export default function AdminRechargeLogs() {
  const [logs, setLogs] = useState<RechargeLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 20;

  const fetchLogs = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const res = await adminApi.listRechargeLogs(page, pageSize);
      setLogs(res.data.data || []);
      setTotal(res.data.total || 0);
    } catch {
      setError('获取充值日志失败');
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
        <h2 className="text-lg font-semibold text-gray-900">充值日志</h2>
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
        <div className="text-center py-12 text-gray-400">暂无充值记录</div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-3 font-medium text-gray-500">时间</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">团队</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">操作人</th>
                  <th className="text-right py-3 px-3 font-medium text-gray-500">充值金额</th>
                  <th className="text-right py-3 px-3 font-medium text-gray-500">充值前余额</th>
                  <th className="text-right py-3 px-3 font-medium text-gray-500">充值后余额</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">IP</th>
                  <th className="text-left py-3 px-3 font-medium text-gray-500">备注</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.id} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3 px-3 text-gray-500 text-xs whitespace-nowrap">{log.created_at}</td>
                    <td className="py-3 px-3 text-gray-900 font-medium">{log.team_name}</td>
                    <td className="py-3 px-3 text-gray-600">{log.operator_name}</td>
                    <td className="py-3 px-3 text-right text-green-600 font-semibold">+¥{formatBalance(log.amount)}</td>
                    <td className="py-3 px-3 text-right text-gray-500">¥{formatBalance(log.balance_before)}</td>
                    <td className="py-3 px-3 text-right text-blue-600 font-medium">¥{formatBalance(log.balance_after)}</td>
                    <td className="py-3 px-3 text-gray-400 font-mono text-xs">{log.ip}</td>
                    <td className="py-3 px-3 text-gray-500 text-xs max-w-[200px] truncate" title={log.remark}>
                      {log.remark || '-'}
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
