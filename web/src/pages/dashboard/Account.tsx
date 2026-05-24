import { useAuth } from '../../hooks/useAuth';

export default function Account() {
  const { user } = useAuth();

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">账户信息</h1>
        <p className="text-gray-500 text-sm mt-1">查看和管理你的账户</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-6 text-sm">
            <div>
              <label className="text-gray-500 block mb-1">用户名</label>
              <p className="text-gray-900 font-medium">{user?.username}</p>
            </div>
            <div>
              <label className="text-gray-500 block mb-1">邮箱</label>
              <p className="text-gray-900 font-medium">{user?.email}</p>
            </div>
            <div>
              <label className="text-gray-500 block mb-1">用户 ID</label>
              <p className="text-gray-900 font-medium">{user?.id}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
