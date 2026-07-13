import { useId, type SVGProps } from 'react'

/**
 * LinuxDO 品牌图标 —— 三色圆形 logo
 * （顶部深色带、中部浅灰带、底部黄色带，外圈浅灰圆环）。
 *
 * 最佳实践：
 * - 外层 span 复刻 @ant-design/icons 的 .anticon 关键样式
 *   （display:inline-flex; align-items:center; line-height:0;
 *   vertical-align:-0.125em），使该图标在 Button / Space / avatar 等
 *   任意容器中都与图标库图标基线对齐、视觉居中，而非仅靠 Button 的
 *   resetIcon() 兜底；
 * - useId() 生成唯一 clipPath id 并剔除冒号，避免同页多实例 id 冲突且
 *   兼容 SVG url(#id) 引用解析；
 * - width/height 用 1em，随宿主 font-size 缩放，与图标库同级图标视觉对齐。
 */
export default function LinuxDoIcon({ className, style, ...rest }: SVGProps<SVGSVGElement>) {
  // React useId() 形如 ":r1:"，冒号在 SVG url(#id) 中需规避
  const clipId = `ld-clip-${useId().replace(/:/g, '')}`

  return (
    <span
      className={className}
      aria-hidden="true"
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        lineHeight: 0,
        verticalAlign: '-0.125em',
        ...style,
      }}
    >
      <svg
        width="1em"
        height="1em"
        viewBox="0 0 120 120"
        xmlns="http://www.w3.org/2000/svg"
        focusable="false"
        {...rest}
      >
        <clipPath id={clipId}>
          <circle cx="60" cy="60" r="47" />
        </clipPath>
        <circle cx="60" cy="60" r="50" fill="#f0f0f0" />
        <rect x="10" y="10" width="100" height="30" fill="#1c1c1e" clipPath={`url(#${clipId})`} />
        <rect x="10" y="40" width="100" height="40" fill="#f0f0f0" clipPath={`url(#${clipId})`} />
        <rect x="10" y="80" width="100" height="30" fill="#ffb003" clipPath={`url(#${clipId})`} />
      </svg>
    </span>
  )
}
