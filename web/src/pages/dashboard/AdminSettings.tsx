import { useState, useEffect, useCallback } from 'react';
import * as adminApi from '../../api/admin';
import type { SiteSetting } from '../../types/api';

export default function AdminSettings() {
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

  const booleanKeys = ['menu_arena_visible', 'menu_docs_visible'];

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">站点配置</h2>
        <button
          onClick={fetchSettings}
          className="inline-flex items-center gap-1.5 rounded-lg bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-200 transition-colors"
        >
          刷新
        </button>
      </div>

      {settingsMsg && (
        <div className={`mb-4 rounded-lg px-4 py-2.5 text-sm ${settingsMsg.type === 'success' ? 'bg-green-50 border border-green-200 text-green-700' : 'bg-red-50 border border-red-200 text-red-600'}`}>
          {settingsMsg.text}
        </div>
      )}

      {settingsLoading ? (
        <div className="flex justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm">
          <div className="divide-y divide-gray-100">
            {settings.map((s) => (
              <div key={s.key} className="flex items-center justify-between px-6 py-4">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900">{s.comment || s.key}</p>
                  <p className="text-xs text-gray-400 mt-0.5">键名: {s.key}</p>
                </div>
                <div className="flex items-center gap-3 ml-4">
                  {editingKey === s.key ? (
                    <>
                      {booleanKeys.includes(s.key) ? (
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
                        {booleanKeys.includes(s.key)
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
        </div>
      )}
    </div>
  );
}
