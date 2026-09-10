import { ArrowLeftOutlined } from '@ant-design/icons';
import { Button } from 'antd';

interface ChainPageHeaderProps {
  title: string;
  onBack: () => void;
}

// a full-width backdrop with the content in the screen body's column: otherwise on desktop
// the heading detaches from the content
export function ChainPageHeader({ title, onBack }: ChainPageHeaderProps) {
  return (
    <header className="chain-detail-page__header">
      <div className="chain-detail-page__header-inner">
        <Button
          className="chain-detail-page__back"
          type="text"
          icon={<ArrowLeftOutlined aria-hidden />}
          aria-label="Назад"
          onClick={onBack}
        />
        <h1 className="chain-detail-page__title">{title}</h1>
      </div>
    </header>
  );
}
