import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Loader2, Download, ThumbsUp, ThumbsDown, BookOpen, CheckCircle2 } from 'lucide-react';
import { annotationsApi, type AnnotationQueueItem } from '../../api/client';
import { useTenant } from '../../contexts/TenantContext';
import { getErrorMessage } from '../../utils/errors';

const ISSUE_TAGS = ['wrong_fact', 'off_topic', 'hallucination', 'format', 'tone', 'incomplete', 'other'];
const ISSUE_LABELS: Record<string, string> = {
  wrong_fact: '事实错误',
  off_topic: '答非所问',
  hallucination: '幻觉',
  format: '格式问题',
  tone: '语气',
  incomplete: '不完整',
  other: '其他',
};

export default function AnnotationsPage() {
  const queryClient = useQueryClient();
  const { activeTenant } = useTenant();
  const tenantId = activeTenant?.tenantId ?? null;

  const [ratingFilter, setRatingFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selected, setSelected] = useState<AnnotationQueueItem | null>(null);

  // Annotation form state.
  const [score, setScore] = useState(3);
  const [tags, setTags] = useState<string[]>([]);
  const [idealAnswer, setIdealAnswer] = useState('');
  const [notes, setNotes] = useState('');
  const [formError, setFormError] = useState<string | null>(null);

  const { data: stats = [] } = useQuery({
    queryKey: ['annotation-stats', tenantId],
    queryFn: () => annotationsApi.stats(),
  });

  const { data: queue = [], isLoading } = useQuery({
    queryKey: ['annotation-queue', tenantId, ratingFilter, statusFilter],
    queryFn: () => annotationsApi.queue({ rating: ratingFilter || undefined, status: statusFilter || undefined }),
  });

  const { data: context = [] } = useQuery({
    queryKey: ['annotation-context', selected?.messageId],
    queryFn: () => annotationsApi.context(selected!.messageId),
    enabled: !!selected,
  });

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['annotation-queue', tenantId] });
    queryClient.invalidateQueries({ queryKey: ['annotation-stats', tenantId] });
  };

  const saveMutation = useMutation({
    mutationFn: () => annotationsApi.save(selected!.messageId, { qualityScore: score, issueTags: tags, idealAnswer, notes }),
    onSuccess: refresh,
    onError: (e) => setFormError(getErrorMessage(e)),
  });

  const promoteMutation = useMutation({
    mutationFn: () => annotationsApi.promote(selected!.messageId),
    onSuccess: () => { refresh(); setFormError(null); },
    onError: (e) => setFormError(getErrorMessage(e)),
  });

  const selectItem = (item: AnnotationQueueItem) => {
    setSelected(item);
    setScore(item.qualityScore ?? 3);
    setTags([]);
    setIdealAnswer('');
    setNotes('');
    setFormError(null);
  };

  const toggleTag = (tag: string) => {
    setTags(prev => prev.includes(tag) ? prev.filter(t => t !== tag) : [...prev, tag]);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-white">数据标注</h1>
          <p className="text-sm text-dark-400">审阅用户反馈、标注质量、沉淀知识、导出训练数据</p>
        </div>
        <a
          href={annotationsApi.exportUrl({ minScore: 4 })}
          className="flex items-center gap-2 rounded-lg border border-white/8 px-4 py-2 text-sm text-dark-200 hover:text-white"
        >
          <Download className="h-4 w-4" /> 导出 JSONL
        </a>
      </div>

      {/* Quality dashboard */}
      {stats.length > 0 && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {stats.map(s => (
            <div key={s.agentId} className="rounded-lg border border-white/8 bg-dark-900 p-3">
              <p className="truncate text-xs text-dark-400">{s.agentId}</p>
              <p className="mt-1 text-lg font-semibold text-white">{s.messageCount} 条</p>
              <div className="mt-1 flex items-center gap-3 text-xs">
                <span className="text-emerald-400">👍 {s.thumbsUp}</span>
                <span className="text-red-400">👎 {s.thumbsDown}</span>
                {s.avgQualityScore > 0 && <span className="text-dark-300">均分 {s.avgQualityScore.toFixed(1)}</span>}
              </div>
              <p className="mt-1 text-xs text-dark-500">
                标注 {s.annotationCount} · 沉淀 {s.promotedCount} · token {(s.promptTokens + s.completionTokens).toLocaleString()}
              </p>
            </div>
          ))}
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        {/* Queue */}
        <div className="rounded-lg border border-white/8 bg-dark-900">
          <div className="flex items-center gap-2 border-b border-white/8 p-3">
            <select value={ratingFilter} onChange={e => setRatingFilter(e.target.value)} className="rounded bg-dark-800 border border-white/8 px-2 py-1 text-sm">
              <option value="">全部评价</option>
              <option value="-1">仅 👎</option>
              <option value="1">仅 👍</option>
            </select>
            <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)} className="rounded bg-dark-800 border border-white/8 px-2 py-1 text-sm">
              <option value="">全部状态</option>
              <option value="unannotated">未标注</option>
              <option value="annotated">已标注</option>
            </select>
          </div>
          <div className="max-h-[60vh] overflow-y-auto p-2">
            {isLoading ? (
              <div className="flex justify-center py-6"><Loader2 className="h-5 w-5 animate-spin text-primary-500" /></div>
            ) : queue.length === 0 ? (
              <p className="py-6 text-center text-sm text-dark-400">队列为空。等待用户反馈或调整筛选。</p>
            ) : (
              <div className="space-y-2">
                {queue.map(item => (
                  <button
                    key={item.messageId}
                    onClick={() => selectItem(item)}
                    className={`w-full rounded-lg border p-3 text-left transition-colors ${selected?.messageId === item.messageId ? 'border-primary-500/60 bg-primary-500/10' : 'border-white/8 hover:border-white/20'}`}
                  >
                    <div className="mb-1 flex items-center gap-2">
                      {item.rating === 1 && <ThumbsUp className="h-3.5 w-3.5 text-emerald-400" />}
                      {item.rating === -1 && <ThumbsDown className="h-3.5 w-3.5 text-red-400" />}
                      <span className="text-xs text-dark-500">{item.agentId}</span>
                      {item.annotated && <CheckCircle2 className="h-3.5 w-3.5 text-primary-400" />}
                      {item.status === 'promoted' && <BookOpen className="h-3.5 w-3.5 text-emerald-400" />}
                    </div>
                    <p className="truncate text-sm text-white">{item.userQuestion || '(无问题)'}</p>
                    <p className="mt-0.5 line-clamp-2 text-xs text-dark-400">{item.assistantReply}</p>
                    {item.comment && <p className="mt-1 text-xs text-amber-300">“{item.comment}”</p>}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Detail / annotation form */}
        <div className="rounded-lg border border-white/8 bg-dark-900 p-4">
          {!selected ? (
            <p className="py-12 text-center text-sm text-dark-400">从左侧选择一条消息进行标注</p>
          ) : (
            <div className="space-y-4">
              {/* Conversation context */}
              <div className="max-h-48 space-y-2 overflow-y-auto rounded-lg border border-white/8 bg-dark-950/40 p-3">
                {context.map(m => (
                  <div key={m.id} className={`text-xs ${m.role === 'assistant' ? 'text-dark-200' : 'text-primary-200'}`}>
                    <span className="font-semibold">{m.role === 'assistant' ? 'AI' : '用户'}:</span> {m.content}
                  </div>
                ))}
              </div>

              {/* Score */}
              <div>
                <label className="mb-1 block text-sm text-dark-400">质量评分(1-5)</label>
                <div className="flex gap-2">
                  {[1, 2, 3, 4, 5].map(n => (
                    <button
                      key={n}
                      onClick={() => setScore(n)}
                      className={`h-9 w-9 rounded-md text-sm ${score === n ? 'bg-primary-500 text-white' : 'bg-dark-800 text-dark-300'}`}
                    >
                      {n}
                    </button>
                  ))}
                </div>
              </div>

              {/* Issue tags */}
              <div>
                <label className="mb-1 block text-sm text-dark-400">问题分类</label>
                <div className="flex flex-wrap gap-2">
                  {ISSUE_TAGS.map(tag => (
                    <button
                      key={tag}
                      onClick={() => toggleTag(tag)}
                      className={`rounded-full px-3 py-1 text-xs ${tags.includes(tag) ? 'bg-primary-500 text-white' : 'bg-dark-800 text-dark-300'}`}
                    >
                      {ISSUE_LABELS[tag]}
                    </button>
                  ))}
                </div>
              </div>

              {/* Ideal answer */}
              <div>
                <label className="mb-1 block text-sm text-dark-400">理想回答(可沉淀为知识)</label>
                <textarea
                  value={idealAnswer}
                  onChange={e => setIdealAnswer(e.target.value)}
                  rows={4}
                  placeholder="写下这条问题应有的标准答案…"
                  className="w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
                />
              </div>

              {/* Notes */}
              <div>
                <label className="mb-1 block text-sm text-dark-400">备注</label>
                <input
                  value={notes}
                  onChange={e => setNotes(e.target.value)}
                  className="w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
                />
              </div>

              {formError && <p className="text-xs text-red-400">{formError}</p>}

              <div className="flex gap-2">
                <button
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                  className="flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm text-white hover:bg-primary-600 disabled:opacity-50"
                >
                  {saveMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                  保存标注
                </button>
                <button
                  onClick={() => promoteMutation.mutate()}
                  disabled={promoteMutation.isPending || !idealAnswer.trim()}
                  className="flex items-center gap-2 rounded-lg border border-emerald-500/30 px-4 py-2 text-sm text-emerald-300 hover:bg-emerald-500/10 disabled:opacity-40"
                  title={!idealAnswer.trim() ? '先填写理想回答' : '沉淀为知识库 QA'}
                >
                  {promoteMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <BookOpen className="h-4 w-4" />}
                  沉淀为知识
                </button>
              </div>
              {saveMutation.isSuccess && <p className="text-xs text-emerald-400">已保存</p>}
              {promoteMutation.isSuccess && <p className="text-xs text-emerald-400">已沉淀到知识库,将在下次对话生效</p>}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
