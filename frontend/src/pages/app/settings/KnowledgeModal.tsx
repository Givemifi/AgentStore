import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Loader2, Trash2, RefreshCw, FileText, Upload, AlertCircle } from 'lucide-react';
import { knowledgeApi, type KnowledgeDocument } from '../../../api/client';
import { prepareAttachment } from '../../../utils/attachments';

interface KnowledgeModalProps {
  agentId: string;
  agentName: string;
  onClose: () => void;
}

type Mode = 'upload' | 'text' | 'qa';

export default function KnowledgeModal({ agentId, agentName, onClose }: KnowledgeModalProps) {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<Mode>('upload');
  const [name, setName] = useState('');
  const [text, setText] = useState('');
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const { data: docs = [], isLoading } = useQuery({
    queryKey: ['knowledge', agentId],
    queryFn: () => knowledgeApi.list(agentId),
    refetchInterval: (query) => {
      const data = query.state.data as KnowledgeDocument[] | undefined;
      // Poll while any document is still processing.
      return data?.some(d => d.status === 'processing') ? 3000 : false;
    },
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['knowledge', agentId] });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; sourceType: string; text: string }) => knowledgeApi.create(agentId, data),
    onSuccess: () => { invalidate(); resetForm(); },
    onError: (e: unknown) => setError(extractError(e)),
  });

  const deleteMutation = useMutation({
    mutationFn: (docId: string) => knowledgeApi.remove(agentId, docId),
    onSuccess: invalidate,
  });

  const reindexMutation = useMutation({
    mutationFn: (docId: string) => knowledgeApi.reindex(agentId, docId),
    onSuccess: invalidate,
  });

  const resetForm = () => {
    setName('');
    setText('');
    setQuestion('');
    setAnswer('');
    setError(null);
  };

  const handleFilePick = async (file: File) => {
    setError(null);
    setBusy(true);
    try {
      const prepared = await prepareAttachment(file);
      if (prepared.kind !== 'document' || !prepared.text) {
        setError('仅支持从 PDF / Word / TXT 文档提取文字');
        return;
      }
      setText(prepared.text);
      if (!name) setName(file.name);
    } catch (e) {
      setError(e instanceof Error ? e.message : '读取文件失败');
    } finally {
      setBusy(false);
    }
  };

  const handleSubmit = () => {
    setError(null);
    if (mode === 'qa') {
      if (!question.trim() || !answer.trim()) {
        setError('问题和答案都不能为空');
        return;
      }
      createMutation.mutate({
        name: name.trim() || question.trim().slice(0, 40),
        sourceType: 'qa',
        text: `问:${question.trim()}\n答:${answer.trim()}`,
      });
      return;
    }
    if (!text.trim()) {
      setError(mode === 'upload' ? '请先选择并解析一个文档' : '内容不能为空');
      return;
    }
    createMutation.mutate({
      name: name.trim() || 'Untitled',
      sourceType: mode === 'upload' ? 'document' : 'text',
      text: text.trim(),
    });
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onClose}>
      <div
        className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl border border-white/8 bg-dark-900 p-6"
        onClick={e => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-white">知识库 · {agentName}</h2>
            <p className="text-sm text-dark-400">上传领域文档让该 Agent 答得更专业</p>
          </div>
          <button onClick={onClose} className="rounded p-1 text-dark-400 hover:bg-dark-800 hover:text-white">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Add form */}
        <div className="mb-6 rounded-lg border border-white/8 p-4">
          <div className="mb-3 flex gap-2">
            {(['upload', 'text', 'qa'] as Mode[]).map(m => (
              <button
                key={m}
                onClick={() => { setMode(m); resetForm(); }}
                className={`rounded-md px-3 py-1.5 text-sm ${mode === m ? 'bg-primary-500 text-white' : 'bg-dark-800 text-dark-300 hover:text-white'}`}
              >
                {m === 'upload' ? '上传文档' : m === 'text' ? '粘贴文本' : '问答对'}
              </button>
            ))}
          </div>

          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="名称(可选)"
            className="mb-3 w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
          />

          {mode === 'upload' && (
            <label className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-dashed border-white/15 bg-dark-950/40 px-4 py-6 text-sm text-dark-300 hover:border-primary-400/40">
              {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
              {text ? '已解析,点击重新选择' : '选择 PDF / Word / TXT 文件'}
              <input
                type="file"
                accept=".pdf,.docx,.txt,.md"
                className="hidden"
                onChange={e => { const f = e.target.files?.[0]; if (f) void handleFilePick(f); }}
              />
            </label>
          )}

          {mode === 'text' && (
            <textarea
              value={text}
              onChange={e => setText(e.target.value)}
              rows={5}
              placeholder="粘贴领域知识文本…"
              className="w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
            />
          )}

          {mode === 'qa' && (
            <div className="space-y-2">
              <input
                value={question}
                onChange={e => setQuestion(e.target.value)}
                placeholder="问题"
                className="w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
              />
              <textarea
                value={answer}
                onChange={e => setAnswer(e.target.value)}
                rows={3}
                placeholder="标准答案"
                className="w-full rounded border border-white/8 bg-dark-800 px-3 py-2 text-sm"
              />
            </div>
          )}

          {error && (
            <div className="mt-3 flex items-start gap-2 rounded-md border border-red-500/20 bg-red-500/10 p-2 text-xs text-red-300">
              <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <button
            onClick={handleSubmit}
            disabled={createMutation.isPending || busy}
            className="mt-3 flex items-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm text-white hover:bg-primary-600 disabled:opacity-50"
          >
            {createMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            添加到知识库
          </button>
        </div>

        {/* Document list */}
        {isLoading ? (
          <div className="flex justify-center py-6"><Loader2 className="h-5 w-5 animate-spin text-primary-500" /></div>
        ) : docs.length === 0 ? (
          <p className="py-6 text-center text-sm text-dark-400">还没有知识文档</p>
        ) : (
          <div className="space-y-2">
            {docs.map(doc => (
              <div key={doc.id} className="flex items-center justify-between rounded-lg border border-white/8 p-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <FileText className="h-4 w-4 shrink-0 text-dark-400" />
                    <span className="truncate text-sm text-white">{doc.name}</span>
                    <StatusBadge status={doc.status} />
                  </div>
                  <p className="mt-1 text-xs text-dark-500">
                    {doc.chunkCount} 块 · {doc.charCount} 字
                    {doc.status === 'error' && doc.errorMessage ? ` · ${doc.errorMessage}` : ''}
                  </p>
                </div>
                <div className="flex shrink-0 gap-1">
                  <button
                    onClick={() => reindexMutation.mutate(doc.id)}
                    disabled={reindexMutation.isPending || doc.status === 'processing'}
                    className="rounded p-1.5 text-dark-400 hover:bg-dark-800 hover:text-white disabled:opacity-40"
                    title="重建索引"
                  >
                    <RefreshCw className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => { if (confirm('删除该知识文档?')) deleteMutation.mutate(doc.id); }}
                    disabled={deleteMutation.isPending}
                    className="rounded p-1.5 text-red-400 hover:bg-dark-800 disabled:opacity-40"
                    title="删除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: KnowledgeDocument['status'] }) {
  const map: Record<KnowledgeDocument['status'], { label: string; cls: string }> = {
    processing: { label: '处理中', cls: 'bg-yellow-500/20 text-yellow-400' },
    ready: { label: '就绪', cls: 'bg-green-500/20 text-green-400' },
    error: { label: '失败', cls: 'bg-red-500/20 text-red-400' },
  };
  const s = map[status];
  return <span className={`rounded px-1.5 py-0.5 text-xs ${s.cls}`}>{s.label}</span>;
}

function extractError(e: unknown): string {
  if (e && typeof e === 'object' && 'response' in e) {
    const resp = (e as { response?: { data?: { error?: string } } }).response;
    if (resp?.data?.error) return resp.data.error;
  }
  return e instanceof Error ? e.message : '操作失败';
}
