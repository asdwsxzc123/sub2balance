import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return '-';
  return new Date(value).toLocaleString();
}

export function formatAmount(value: number | null | undefined): string {
  if (value === null || value === undefined) return '$0.00';
  return `$${value.toFixed(2)}`;
}

const CURRENCY_SYMBOL: Record<string, string> = {
  USD: '$',
  CNY: '¥',
};

export function formatMoney(value: number | null | undefined, currency?: string | null): string {
  const symbol = CURRENCY_SYMBOL[(currency || 'USD').toUpperCase()] ?? '';
  const n = value == null ? 0 : value;
  return `${symbol}${n.toFixed(2)}`;
}
