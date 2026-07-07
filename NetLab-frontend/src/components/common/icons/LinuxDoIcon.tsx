import type { SVGProps } from 'react'

/**
 * LinuxDO 品牌图标 —— 灵感源自三色 logo 的几何标识
 * （深色背景上的黄色三角形，寓意“三分黑暗，七分光明”）。
 */
export default function LinuxDoIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      width="1em"
      height="1em"
      viewBox="0 0 32 32"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      {/* 深色圆形背景 */}
      <circle cx="16" cy="16" r="15" fill="#1A1A2E" stroke="currentColor" strokeWidth="1.5" />
      {/* 黄色几何点缀 —— 风格化三角形 */}
      <path
        d="M10 22L16 8L22 22H10Z"
        fill="#F5C542"
        opacity="0.9"
      />
      {/* 内部镂空以增强层次感 */}
      <path
        d="M13.5 20L16 13L18.5 20H13.5Z"
        fill="#1A1A2E"
        opacity="0.7"
      />
    </svg>
  )
}
