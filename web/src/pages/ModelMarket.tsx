import { useState, useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import PublicNavbar from '../components/PublicNavbar';
import { useAuth } from '../hooks/useAuth';
import client from '../api/client';

/* ---------- 后端 pricing 接口返回类型 ---------- */
interface PricingVendor {
  id: number;
  name: string;
  icon: string;
}

interface PricingItem {
  model_name: string;
  vendor_id: number;
  quota_type: number;
  model_ratio: number;
  model_price: number;
  completion_ratio: number;
  owner_by: string;
  enable_groups: string[];
  supported_endpoint_types: string[];
}

interface PricingResponse {
  success: boolean;
  data: PricingItem[];
  vendors: PricingVendor[];
  group_ratio: Record<string, number>;
  auto_groups: string[];
}

/* ---------- 前端展示用模型类型 ---------- */
interface ModelCard {
  id: string;
  name: string;
  provider: string;
  providerIcon: string;
  category: string;
  description: string;
  contextWindow: string;
  pricing: { input: string; output: string };
  tags: string[];
  featured: boolean;
}

/* ---------- 供应商颜色映射 ---------- */
const providerColorMap: Record<string, string> = {
  OpenAI: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  Anthropic: 'bg-orange-50 text-orange-700 border-orange-200',
  DeepSeek: 'bg-blue-50 text-blue-700 border-blue-200',
  Google: 'bg-red-50 text-red-700 border-red-200',
  阿里云: 'bg-purple-50 text-purple-700 border-purple-200',
  'Stability AI': 'bg-indigo-50 text-indigo-700 border-indigo-200',
  Midjourney: 'bg-pink-50 text-pink-700 border-pink-200',
  BAAI: 'bg-teal-50 text-teal-700 border-teal-200',
};

function providerBadge(p: string) {
  const color = providerColorMap[p] || 'bg-gray-50 text-gray-600 border-gray-200';
  return (
    <span className={`text-xs font-medium px-2 py-0.5 rounded-full border ${color}`}>
      {p}
    </span>
  );
}

/* ---------- 根据模型名推断分类 ---------- */
function guessCategory(name: string): string {
  const lower = name.toLowerCase();
  if (lower.includes('embedding') || lower.includes('bge')) return '嵌入';
  if (lower.includes('tts') || lower.includes('whisper') || lower.includes('audio') || lower.includes('speech')) return '语音';
  if (lower.includes('dall-e') || lower.includes('stable-diffusion') || lower.includes('midjourney') || lower.includes('flux') || lower.includes('image')) return '图片';
  return '对话';
}

/* ---------- 根据模型名推断描述 ---------- */
function guessDescription(name: string, provider: string): string {
  const lower = name.toLowerCase();
  if (lower.includes('gpt-4o')) return 'OpenAI 最新的多模态旗舰模型，支持文本、图像、音频输入，具有极高的推理能力与响应速度。';
  if (lower.includes('gpt-4-turbo') || lower.includes('gpt-4-1106')) return '高性能推理模型，针对速度和成本进行了优化，适合复杂的分析与创作任务。';
  if (lower.includes('claude-3.5') || lower.includes('claude-3-5')) return 'Anthropic 的中高端模型，在代码生成、长篇写作和复杂推理方面表现卓越。';
  if (lower.includes('claude-3-opus') || lower.includes('claude-3-opus')) return 'Anthropic 的旗舰模型，在复杂分析、研究与深度推理方面具有最强表现。';
  if (lower.includes('claude')) return 'Anthropic 的大语言模型，擅长深度推理与安全可控的对话。';
  if (lower.includes('deepseek-r1')) return '专注推理能力的开源模型，通过强化学习在数学、编程和逻辑推理方面达到顶尖水平。';
  if (lower.includes('deepseek-v3')) return '国产开源大模型的标杆，671B MOE 架构，综合能力比肩 GPT-4，价格极具竞争力。';
  if (lower.includes('deepseek')) return 'DeepSeek 系列大语言模型，高性价比、开源可商用。';
  if (lower.includes('gemini-2.0-flash') || lower.includes('gemini-2.0-flash')) return 'Google 最新轻量级多模态模型，极低延迟，支持百万级上下文，适合实时交互场景。';
  if (lower.includes('gemini-2.0-pro') || lower.includes('gemini-2.0-pro')) return 'Google 最新旗舰模型，在编程、推理和多模态理解方面表现全面。';
  if (lower.includes('gemini')) return 'Google 多模态大模型系列，支持文本、图像、音频、视频输入。';
  if (lower.includes('qwen')) return `${provider} 千问系列模型，中英文双语能力突出，企业级推理性能。`;
  if (lower.includes('glm')) return `${provider} 智谱 GLM 系列模型，支持中英双语对话与工具调用。`;
  if (lower.includes('dall-e')) return '文本到图像生成模型，能精确理解复杂指令，生成高品质图片。';
  if (lower.includes('stable-diffusion')) return '开源图片生成模型，支持高分辨率输出，灵活可控的风格与构图。';
  if (lower.includes('midjourney')) return '艺术风格出类拔萃的图片生成服务，适合创意设计和视觉创作。';
  if (lower.includes('embedding') || lower.includes('bge')) return '语义嵌入模型，适合语义搜索、聚类、推荐等向量相关任务。';
  if (lower.includes('whisper')) return '高精度通用语音识别模型，支持近百种语言转写与翻译。';
  if (lower.includes('tts')) return '文本转语音模型，支持多种音色和自然语调。';
  return `${provider} 提供的 ${name} 模型。`;
}

/* ---------- 根据模型名推断上下文窗口 ---------- */
function guessContextWindow(name: string): string {
  const lower = name.toLowerCase();
  if (lower.includes('gemini')) return '1M+';
  if (lower.includes('claude-3') || lower.includes('claude-3.5')) return '200K';
  if (lower.includes('gpt-4') || lower.includes('gpt-3.5') || lower.includes('deepseek')) return '128K';
  if (lower.includes('claude')) return '200K';
  if (lower.includes('qwen')) return '32K';
  if (lower.includes('glm')) return '128K';
  return '128K';
}

/* ---------- 转换 pricing 数据为前端展示格式 ---------- */
function toModelCard(item: PricingItem, vendors: PricingVendor[]): ModelCard {
  const vendor = vendors.find((v) => v.id === item.vendor_id);
  const provider = vendor?.name || '未知';
  const category = guessCategory(item.model_name);

  const inputPrice = item.model_ratio > 0
    ? `¥${(item.model_ratio * 0.01).toFixed(4)}/1K tokens`
    : '按量计费';
  const outputPrice = item.completion_ratio > 0
    ? `¥${(item.completion_ratio * 0.01).toFixed(4)}/1K tokens`
    : item.model_ratio > 0 ? inputPrice : '按量计费';

  const tags: string[] = [];
  if (item.supported_endpoint_types) {
    tags.push(...item.supported_endpoint_types);
  }
  if (provider) tags.push(provider);

  // 前几个模型标记为推荐
  const featuredModels = ['gpt-4o', 'claude-3.5-sonnet', 'deepseek-v3', 'deepseek-r1', 'gemini-2.0-flash'];
  const featured = featuredModels.some((m) => item.model_name.toLowerCase().includes(m.toLowerCase()));

  return {
    id: item.model_name,
    name: item.model_name,
    provider,
    providerIcon: vendor?.icon || '',
    category,
    description: guessDescription(item.model_name, provider),
    contextWindow: guessContextWindow(item.model_name),
    pricing: { input: inputPrice, output: outputPrice },
    tags,
    featured,
  };
}

const categories = ['全部', '对话', '图片', '嵌入', '语音'];

export default function ModelMarket() {
  const { user } = useAuth();
  const [models, setModels] = useState<ModelCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState('全部');
  const [selectedProvider, setSelectedProvider] = useState('全部');

  // 加载数据
  useEffect(() => {
    client.get<PricingResponse>('/pricing')
      .then((res) => {
        if (res.data?.data && res.data?.vendors) {
          const cards = res.data.data
            .filter((item) => item.model_name) // 过滤空模型名
            .map((item) => toModelCard(item, res.data.vendors));
          setModels(cards);
        }
      })
      .catch((err) => {
        console.error('获取定价数据失败:', err);
        setError('加载模型列表失败，请稍后重试');
      })
      .finally(() => setLoading(false));
  }, []);

  const providers = useMemo(() => {
    const all = [...new Set(models.map((m) => m.provider))];
    return ['全部', ...all.sort()];
  }, [models]);

  const filtered = useMemo(() => {
    return models.filter((m) => {
      if (activeCategory !== '全部' && m.category !== activeCategory) return false;
      if (selectedProvider !== '全部' && m.provider !== selectedProvider) return false;
      if (search) {
        const q = search.toLowerCase();
        return (
          m.name.toLowerCase().includes(q) ||
          m.provider.toLowerCase().includes(q) ||
          m.description.toLowerCase().includes(q) ||
          m.tags.some((t) => t.toLowerCase().includes(q))
        );
      }
      return true;
    });
  }, [activeCategory, selectedProvider, search, models]);

  return (
    <div className="min-h-screen bg-gray-50">
      <PublicNavbar />

      {/* Header */}
      <section className="bg-white border-b border-gray-200">
        <div className="max-w-[1440px] mx-auto px-6 py-16">
          <h1 className="text-3xl sm:text-4xl font-extrabold text-gray-900">模型广场</h1>
          <p className="mt-3 text-lg text-gray-500 max-w-xl">
            浏览 ModelGate 接入的全部 AI 模型，按分类和供应商筛选，找到最适合你的模型。
          </p>

          {/* Search & Filters */}
          <div className="mt-8 space-y-4">
            <div className="relative max-w-md">
              <svg
                className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400"
                fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
              </svg>
              <input
                type="text"
                placeholder="搜索模型名称、供应商或功能..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-11 pr-4 py-3 rounded-xl border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-orange-500 focus:border-transparent"
              />
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              {categories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setActiveCategory(cat)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                    activeCategory === cat
                      ? 'bg-orange-500 text-white shadow-sm'
                      : 'bg-white text-gray-600 border border-gray-200 hover:border-gray-300 hover:text-gray-900'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs text-gray-400 mr-1">供应商:</span>
              {providers.map((p) => (
                <button
                  key={p}
                  onClick={() => setSelectedProvider(p)}
                  className={`text-xs px-2.5 py-1 rounded-full border transition-all ${
                    selectedProvider === p
                      ? 'border-orange-500 bg-orange-50 text-orange-700'
                      : 'border-gray-200 bg-white text-gray-500 hover:border-gray-300'
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* Model Grid */}
      <section className="max-w-[1440px] mx-auto px-6 py-12">
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <div className="flex flex-col items-center gap-3">
              <svg className="animate-spin w-8 h-8 text-orange-500" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <span className="text-sm text-gray-400">加载模型列表...</span>
            </div>
          </div>
        ) : error ? (
          <div className="text-center py-24">
            <p className="text-gray-400 text-lg">{error}</p>
            <button
              onClick={() => window.location.reload()}
              className="mt-4 text-sm text-orange-500 hover:text-orange-600 font-medium"
            >
              重新加载
            </button>
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-24">
            <p className="text-gray-400 text-lg">
              {models.length === 0 ? '暂无可用的模型' : '没有找到匹配的模型'}
            </p>
            {models.length > 0 && (
              <button
                onClick={() => { setSearch(''); setActiveCategory('全部'); setSelectedProvider('全部'); }}
                className="mt-4 text-sm text-orange-500 hover:text-orange-600 font-medium"
              >
                清除所有筛选条件
              </button>
            )}
          </div>
        ) : (
          <>
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {filtered.map((m) => (
                <div
                  key={m.id}
                  className="group relative bg-white rounded-2xl border border-gray-100 p-6 hover:border-orange-200 hover:shadow-lg hover:shadow-orange-500/5 transition-all duration-300"
                >
                  {m.featured && (
                    <div className="absolute -top-2 -right-2">
                      <span className="inline-flex items-center gap-1 rounded-full bg-gradient-to-r from-orange-400 to-red-500 px-2.5 py-0.5 text-[10px] font-semibold text-white shadow-sm">
                        推荐
                      </span>
                    </div>
                  )}

                  <div className="flex items-center gap-2 mb-3">
                    {providerBadge(m.provider)}
                    <span className="text-xs text-gray-400">{m.category}</span>
                  </div>

                  <h3 className="text-lg font-semibold text-gray-900 mb-2 group-hover:text-orange-600 transition-colors">
                    {m.name}
                  </h3>

                  <p className="text-sm text-gray-500 leading-relaxed mb-4 line-clamp-2">
                    {m.description}
                  </p>

                  <div className="flex items-center gap-4 text-xs text-gray-400 mb-4">
                    {m.contextWindow !== '-' && (
                      <span className="inline-flex items-center gap-1">
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" strokeWidth={1.5} viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M6 6.878V6a2.25 2.25 0 012.25-2.25h7.5A2.25 2.25 0 0118 6v.878m-12 0c.235-.083.487-.128.75-.128h10.5c.263 0 .515.045.75.128m-12 0A2.25 2.25 0 004.5 9v.878m13.5-3A2.25 2.25 0 0119.5 9v.878m0 0a2.246 2.246 0 00-.75-.128H5.25c-.263 0-.515.045-.75.128m15 0A2.25 2.25 0 0121 12v6a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 18v-6c0-.98.626-1.813 1.5-2.122" />
                        </svg>
                        {m.contextWindow}
                      </span>
                    )}
                    <span>输入 {m.pricing.input}</span>
                  </div>

                  <div className="flex items-center gap-1.5 flex-wrap">
                    {m.tags.slice(0, 5).map((t) => (
                      <span key={t} className="text-[11px] bg-gray-50 text-gray-500 px-2 py-0.5 rounded-md">
                        {t}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-16 text-center">
              <p className="text-sm text-gray-400">
                以上模型及其定价信息实时同步自底层网关，实际调用费用以 API 结算为准。
              </p>
              {!user && (
                <Link
                  to="/register"
                  className="inline-flex items-center justify-center mt-6 rounded-xl bg-gradient-to-r from-orange-500 to-red-500 px-8 py-3 text-sm font-semibold text-white hover:from-orange-600 hover:to-red-600 transition-all shadow-lg shadow-orange-500/25"
                >
                  注册并开始使用
                </Link>
              )}
            </div>
          </>
        )}
      </section>

      <footer className="border-t border-gray-200 bg-white mt-16">
        <div className="max-w-[1440px] mx-auto px-6 py-8 text-center text-sm text-gray-400">
          © {new Date().getFullYear()} ModelGate. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
