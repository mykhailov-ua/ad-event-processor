import type { InputHTMLAttributes } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './field_input.module.css';

export type FieldInputProps = InputHTMLAttributes<HTMLInputElement>;

export function FieldInput({ className, ...rest }: FieldInputProps) {
  return <input className={cn(styles.root, className)} {...rest} />;
}
