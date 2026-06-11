import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Save, Loader2, CreditCard, ShieldAlert, CheckCircle2, Circle } from 'lucide-react';
import { toast } from 'sonner';
import { adminApi } from '../../api/client';
import LoadingSpinner from '../../components/LoadingSpinner';

const MASKED = '••••••';

function ConfiguredBadge({ configured }: { configured: boolean }) {
  return configured ? (
    <span className="flex items-center gap-1 text-xs font-medium text-emerald-400">
      <CheckCircle2 className="w-3.5 h-3.5" /> 已配置
    </span>
  ) : (
    <span className="flex items-center gap-1 text-xs font-medium text-dark-500">
      <Circle className="w-3.5 h-3.5" /> 未配置
    </span>
  );
}

function Field({
  label, value, onChange, placeholder, type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: string;
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-dark-300 mb-2">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full px-4 py-3 bg-dark-800 border border-white/8 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500"
      />
    </div>
  );
}

// ── WeChat Pay Section ───────────────────────────────────────────────────────

interface WechatForm {
  appId: string;
  mchId: string;
  apiV3Key: string;
  privateKey: string;
  certSerialNo: string;
  notifyUrl: string;
  enabled: boolean;
}

function WechatSection() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<WechatForm>({
    appId: '', mchId: '', apiV3Key: '', privateKey: '',
    certSerialNo: '', notifyUrl: '', enabled: false,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['payment-config', 'wechat'],
    queryFn: () => adminApi.getPaymentConfig('wechat'),
  });

  useEffect(() => {
    if (data) {
      setForm({
        appId:        String(data.appId ?? ''),
        mchId:        String(data.mchId ?? ''),
        apiV3Key:     data.apiV3Key === MASKED ? '' : String(data.apiV3Key ?? ''),
        privateKey:   data.privateKey === MASKED ? '' : String(data.privateKey ?? ''),
        certSerialNo: String(data.certSerialNo ?? ''),
        notifyUrl:    String(data.notifyUrl ?? ''),
        enabled:      Boolean(data.enabled),
      });
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: (d: WechatForm) => adminApi.updatePaymentConfig('wechat', d as unknown as Record<string, unknown>),
    onSuccess: (res) => {
      if (res.warning) {
        toast.warning('已保存，但服务初始化失败：' + res.warning);
      } else {
        toast.success('微信支付配置已保存');
      }
      queryClient.invalidateQueries({ queryKey: ['payment-config', 'wechat'] });
    },
    onError: () => toast.error('保存失败'),
  });

  if (isLoading) return <LoadingSpinner size="sm" className="py-8" />;

  return (
    <div className="bg-dark-900/50 border border-white/8 rounded-xl p-6">
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <CreditCard className="w-5 h-5 text-primary-400" />
          微信支付 (WeChat Pay H5)
        </h2>
        <ConfiguredBadge configured={Boolean(data?.configured)} />
      </div>

      <form onSubmit={(e) => { e.preventDefault(); mutation.mutate(form); }} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="App ID" value={form.appId} onChange={v => setForm(f => ({ ...f, appId: v }))} placeholder="wx_xxxxxxxxxxxxxxxxxx" />
          <Field label="商户号 (Mch ID)" value={form.mchId} onChange={v => setForm(f => ({ ...f, mchId: v }))} placeholder="1234567890" />
        </div>
        <Field
          label="API v3 密钥"
          type="password"
          value={form.apiV3Key}
          onChange={v => setForm(f => ({ ...f, apiV3Key: v }))}
          placeholder={data?.apiV3Key === MASKED ? '已保存 (不修改请留空)' : '32位字符'}
        />
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">商户私钥 PEM</label>
          <textarea
            rows={4}
            value={form.privateKey}
            onChange={e => setForm(f => ({ ...f, privateKey: e.target.value }))}
            placeholder={data?.privateKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN RSA PRIVATE KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-white/8 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <Field label="证书序列号" value={form.certSerialNo} onChange={v => setForm(f => ({ ...f, certSerialNo: v }))} placeholder="ABCDEF123456..." />
        <Field label="回调通知 URL" value={form.notifyUrl} onChange={v => setForm(f => ({ ...f, notifyUrl: v }))} placeholder="https://yourdomain.com/api/billing/wechat/notify" />

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
            className="w-5 h-5 rounded border-white/8 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">启用微信支付</span>
        </label>

        <button
          type="submit"
          disabled={mutation.isPending}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          保存微信支付配置
        </button>
      </form>
    </div>
  );
}

// ── Alipay Section ───────────────────────────────────────────────────────────

interface AlipayForm {
  appId: string;
  privateKey: string;
  publicKey: string;
  notifyUrl: string;
  returnUrl: string;
  isSandbox: boolean;
  enabled: boolean;
}

function AlipaySection() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<AlipayForm>({
    appId: '', privateKey: '', publicKey: '',
    notifyUrl: '', returnUrl: '', isSandbox: true, enabled: false,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['payment-config', 'alipay'],
    queryFn: () => adminApi.getPaymentConfig('alipay'),
  });

  useEffect(() => {
    if (data) {
      setForm({
        appId:      String(data.appId ?? ''),
        privateKey: data.privateKey === MASKED ? '' : String(data.privateKey ?? ''),
        publicKey:  data.publicKey === MASKED ? '' : String(data.publicKey ?? ''),
        notifyUrl:  String(data.notifyUrl ?? ''),
        returnUrl:  String(data.returnUrl ?? ''),
        isSandbox:  Boolean(data.isSandbox ?? true),
        enabled:    Boolean(data.enabled),
      });
    }
  }, [data]);

  const mutation = useMutation({
    mutationFn: (d: AlipayForm) => adminApi.updatePaymentConfig('alipay', d as unknown as Record<string, unknown>),
    onSuccess: (res) => {
      if (res.warning) {
        toast.warning('已保存，但服务初始化失败：' + res.warning);
      } else {
        toast.success('支付宝配置已保存');
      }
      queryClient.invalidateQueries({ queryKey: ['payment-config', 'alipay'] });
    },
    onError: () => toast.error('保存失败'),
  });

  if (isLoading) return <LoadingSpinner size="sm" className="py-8" />;

  return (
    <div className="bg-dark-900/50 border border-white/8 rounded-xl p-6">
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <CreditCard className="w-5 h-5 text-blue-400" />
          支付宝 (Alipay)
        </h2>
        <ConfiguredBadge configured={Boolean(data?.configured)} />
      </div>

      <form onSubmit={(e) => { e.preventDefault(); mutation.mutate(form); }} className="space-y-4">
        <Field label="App ID" value={form.appId} onChange={v => setForm(f => ({ ...f, appId: v }))} placeholder="2021000000000000" />
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">应用私钥 PEM</label>
          <textarea
            rows={4}
            value={form.privateKey}
            onChange={e => setForm(f => ({ ...f, privateKey: e.target.value }))}
            placeholder={data?.privateKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN RSA PRIVATE KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-white/8 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-dark-300 mb-2">支付宝公钥 PEM</label>
          <textarea
            rows={4}
            value={form.publicKey}
            onChange={e => setForm(f => ({ ...f, publicKey: e.target.value }))}
            placeholder={data?.publicKey === MASKED ? '已保存 (不修改请留空)' : '-----BEGIN PUBLIC KEY-----\n...'}
            className="w-full px-4 py-3 bg-dark-800 border border-white/8 rounded-lg text-white placeholder-dark-500 focus:outline-none focus:border-primary-500 font-mono text-xs resize-none"
          />
        </div>
        <Field label="异步通知 URL" value={form.notifyUrl} onChange={v => setForm(f => ({ ...f, notifyUrl: v }))} placeholder="https://yourdomain.com/api/billing/alipay/notify" />
        <Field label="同步跳转 URL" value={form.returnUrl} onChange={v => setForm(f => ({ ...f, returnUrl: v }))} placeholder="https://yourdomain.com/billing/success" />

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.isSandbox}
            onChange={e => setForm(f => ({ ...f, isSandbox: e.target.checked }))}
            className="w-5 h-5 rounded border-white/8 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">沙箱模式 (开发测试用)</span>
        </label>

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={form.enabled}
            onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
            className="w-5 h-5 rounded border-white/8 bg-dark-800 text-primary-500 focus:ring-primary-500"
          />
          <span className="text-dark-300 text-sm">启用支付宝</span>
        </label>

        <button
          type="submit"
          disabled={mutation.isPending}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
        >
          {mutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          保存支付宝配置
        </button>
      </form>
    </div>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function PaymentConfigPage() {
  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <CreditCard className="w-7 h-7 text-primary-400" />
          Payment Config
        </h1>
        <p className="text-dark-400 mt-1">
          配置国内支付渠道凭证，保存后立即生效（无需重启）
        </p>
      </div>

      <div className="space-y-6">
        <WechatSection />
        <AlipaySection />
      </div>

      <div className="mt-6 p-4 rounded-xl bg-dark-900/30 border border-white/8 text-xs text-dark-500 flex items-start gap-2">
        <ShieldAlert className="w-3.5 h-3.5 mt-0.5 shrink-0" />
        <span>
          私钥等敏感字段在显示时已脱敏。留空字段保存时不会覆盖原值。订阅套餐计费仍走 Stripe，此处仅管理积分包的国内支付渠道。
        </span>
      </div>
    </div>
  );
}
