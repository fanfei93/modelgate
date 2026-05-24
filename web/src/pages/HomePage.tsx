import { Link } from 'react-router-dom';
import PublicNavbar from '../components/PublicNavbar';

const features = [
  {
    title: '模型广场',
    desc: '汇聚 GPT、Claude、DeepSeek、Gemini 等主流大模型，一站式浏览与对比。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 00-2.455 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z" />
      </svg>
    ),
  },
  {
    title: '统一 API',
    desc: '一套 API 接入所有模型，无需为每个厂商维护不同的 SDK 和密钥。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
      </svg>
    ),
  },
  {
    title: '团队协作',
    desc: '创建团队，邀请成员，共享 API 额度，告别个人账户混乱管理。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" />
      </svg>
    ),
  },
  {
    title: '用量监控',
    desc: '实时查看 API 调用日志，按团队成员和模型维度分析使用情况。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
      </svg>
    ),
  },
  {
    title: '额度管理',
    desc: '统一充值、团队共享额度，清晰透明的消费记录。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" />
      </svg>
    ),
  },
  {
    title: '安全可靠',
    desc: 'HTTPS 加密传输，API Key 安全管理，完善的权限控制体系。',
    icon: (
      <svg className="w-6 h-6" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z" />
      </svg>
    ),
  },
];

const stats = [
  { value: '100+', label: 'AI 模型' },
  { value: '10ms', label: '平均延迟' },
  { value: '99.9%', label: '服务可用性' },
  { value: '24/7', label: '技术支持' },
];

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white">
      <PublicNavbar />

      {/* Hero */}
      <section className="relative overflow-hidden">
        {/* Background decoration */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute -top-40 -right-40 w-80 h-80 rounded-full bg-gradient-to-br from-orange-100 to-red-100 opacity-60 blur-3xl" />
          <div className="absolute -bottom-40 -left-40 w-96 h-96 rounded-full bg-gradient-to-tr from-blue-100 to-purple-100 opacity-50 blur-3xl" />
        </div>

        <div className="relative max-w-[1440px] mx-auto px-6 py-28 text-center">
          <div className="inline-flex items-center gap-2 rounded-full border border-orange-200 bg-orange-50 px-4 py-1.5 text-sm font-medium text-orange-700 mb-8">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-orange-400 opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-orange-500" />
            </span>
            现已支持 GPT-4o、Claude 3.5 等 100+ 模型
          </div>

          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-extrabold text-gray-900 tracking-tight leading-tight">
            一站式{' '}
            <span className="text-gradient">AI API 网关</span>
          </h1>
          <p className="mt-6 text-lg sm:text-xl text-gray-600 max-w-2xl mx-auto leading-relaxed">
            ModelGate 为你提供统一的 AI 模型接入入口。管理团队、监控用量、控制成本，
            让 AI 开发更高效、更安全。
          </p>
          <div className="mt-10 flex items-center justify-center gap-4 flex-wrap">
            <Link
              to="/register"
              className="inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-orange-500 to-red-500 px-8 py-3.5 text-sm font-semibold text-white hover:from-orange-600 hover:to-red-600 transition-all shadow-lg shadow-orange-500/25 hover:shadow-orange-500/40"
            >
              免费开始使用
              <svg className="w-4 h-4 ml-2" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3" />
              </svg>
            </Link>
            <Link
              to="/models"
              className="inline-flex items-center justify-center rounded-xl border-2 border-gray-200 px-8 py-3.5 text-sm font-semibold text-gray-700 hover:border-gray-300 hover:bg-gray-50 transition-all"
            >
              浏览模型广场
            </Link>
          </div>
        </div>
      </section>

      {/* Stats */}
      <section className="border-y border-gray-100 bg-gray-50/50">
        <div className="max-w-[1440px] mx-auto px-6 py-12">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-8">
            {stats.map((s) => (
              <div key={s.label} className="text-center">
                <div className="text-3xl font-extrabold text-gray-900">{s.value}</div>
                <div className="mt-1 text-sm text-gray-500">{s.label}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="max-w-[1440px] mx-auto px-6 py-24">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">为什么选择 ModelGate？</h2>
          <p className="mt-4 text-lg text-gray-500 max-w-xl mx-auto">
            为 AI 开发者和团队打造的全栈式 API 管理平台
          </p>
        </div>
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f, i) => (
            <div
              key={i}
              className="group relative bg-white rounded-2xl border border-gray-100 p-8 hover:border-orange-100 hover:shadow-lg hover:shadow-orange-500/5 transition-all duration-300"
            >
              <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-orange-50 to-red-50 text-orange-600 flex items-center justify-center mb-5 group-hover:scale-110 transition-transform">
                {f.icon}
              </div>
              <h3 className="text-lg font-semibold text-gray-900 mb-2">{f.title}</h3>
              <p className="text-sm text-gray-500 leading-relaxed">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How it works */}
      <section className="bg-gray-50/50 border-y border-gray-100">
        <div className="max-w-[1440px] mx-auto px-6 py-24">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">三步开始</h2>
            <p className="mt-4 text-lg text-gray-500">简单快速，即开即用</p>
          </div>
          <div className="grid gap-8 sm:grid-cols-3 max-w-4xl mx-auto">
            {[
              { step: '01', title: '注册账户', desc: '创建 ModelGate 账户，获取 API 访问凭证。' },
              { step: '02', title: '创建团队', desc: '建立团队，邀请成员，统一管理 API 额度。' },
              { step: '03', title: '开始调用', desc: '使用统一 API 端点，接入所有需要的 AI 模型。' },
            ].map((s) => (
              <div key={s.step} className="text-center">
                <div className="w-16 h-16 mx-auto rounded-full bg-gradient-to-br from-orange-500 to-red-500 flex items-center justify-center text-white text-xl font-bold shadow-lg shadow-orange-500/25">
                  {s.step}
                </div>
                <h3 className="mt-6 text-lg font-semibold text-gray-900">{s.title}</h3>
                <p className="mt-2 text-sm text-gray-500">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="max-w-[1440px] mx-auto px-6 py-24 text-center">
        <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">准备好开始了吗？</h2>
        <p className="mt-4 text-lg text-gray-500">免费注册，即刻体验统一的 AI API 网关服务。</p>
        <div className="mt-8">
          <Link
            to="/register"
            className="inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-orange-500 to-red-500 px-10 py-4 text-base font-semibold text-white hover:from-orange-600 hover:to-red-600 transition-all shadow-lg shadow-orange-500/25"
          >
            免费注册
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-gray-100 bg-white">
        <div className="max-w-[1440px] mx-auto px-6 py-10">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
            <Link to="/" className="text-lg font-bold">
              <span className="bg-gradient-to-r from-orange-400 to-red-500 bg-clip-text text-transparent">
                ModelGate
              </span>
            </Link>
            <div className="flex items-center gap-6 text-sm text-gray-500">
              <Link to="/models" className="hover:text-gray-900 transition-colors">模型广场</Link>
              <Link to="/login" className="hover:text-gray-900 transition-colors">登录</Link>
              <Link to="/register" className="hover:text-gray-900 transition-colors">注册</Link>
            </div>
          </div>
          <div className="mt-8 pt-6 border-t border-gray-100 text-center text-sm text-gray-400">
            © {new Date().getFullYear()} ModelGate. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  );
}
