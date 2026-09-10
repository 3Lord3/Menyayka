import { Button } from 'antd';

import './DealSuccessModal.scss';

interface DealSuccessModalProps {
  emoji: string;
  title: string;
  text: string;
  onClose: () => void;
}

// antd-Modal isn't imported (no-restricted-imports), so the heading lives in the content
export function DealSuccessModal({ emoji, title, text, onClose }: DealSuccessModalProps) {
  return (
    <div className="deal-success">
      <p className="deal-success__icon" aria-hidden>
        {emoji}
      </p>
      <p className="deal-success__title">{title}</p>
      <p className="deal-success__text">{text}</p>
      <Button className="deal-success__button" type="primary" block onClick={onClose}>
        Понятно
      </Button>
    </div>
  );
}
