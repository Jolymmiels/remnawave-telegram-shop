import YookassaIcon from '../../assets/icons/yookassa.svg';
import TributeIcon from '../../assets/icons/tribute.svg';
import CryptoBotIcon from '../../assets/icons/CryptoBot.svg';
import TelegramStarIcon from '../../assets/icons/telegram-star.svg';

interface PaymentIconProps {
  size?: number;
  className?: string;
}

export const YookassaLogo = ({ size = 24, className }: PaymentIconProps) => (
  <img src={YookassaIcon} alt="YooKassa" width={size} height={size} className={className} style={{ objectFit: 'contain' }} />
);

export const TributeLogo = ({ size = 24, className }: PaymentIconProps) => (
  <img src={TributeIcon} alt="Tribute" width={size} height={size} className={className} />
);

export const CryptoPayLogo = ({ size = 24, className }: PaymentIconProps) => (
  <img src={CryptoBotIcon} alt="CryptoPay" width={size} height={size} className={className} />
);

export const TelegramStarsLogo = ({ size = 24, className }: PaymentIconProps) => (
  <img src={TelegramStarIcon} alt="Telegram Stars" width={size} height={size} className={className} />
);

type PaymentType = 'yookassa' | 'crypto' | 'telegram' | 'tribute';

interface PaymentIconByTypeProps extends PaymentIconProps {
  type: PaymentType;
}

export const PaymentIcon = ({ type, size = 24, className }: PaymentIconByTypeProps) => {
  switch (type) {
    case 'yookassa':
      return <YookassaLogo size={size} className={className} />;
    case 'crypto':
      return <CryptoPayLogo size={size} className={className} />;
    case 'telegram':
      return <TelegramStarsLogo size={size} className={className} />;
    case 'tribute':
      return <TributeLogo size={size} className={className} />;
    default:
      return null;
  }
};

export const getPaymentLabel = (type: PaymentType): string => {
  switch (type) {
    case 'yookassa':
      return 'YooKassa';
    case 'crypto':
      return 'CryptoPay';
    case 'telegram':
      return 'Telegram Stars';
    case 'tribute':
      return 'Tribute';
    default:
      return type;
  }
};
