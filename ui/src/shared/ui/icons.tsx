import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement>;

function Base({ children, ...props }: IconProps) {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      {children}
    </svg>
  );
}

export function IconSearch(props: IconProps) {
  return (
    <Base {...props}>
      <circle cx="9" cy="9" r="5.5" />
      <path d="M13.5 13.5 17 17" />
    </Base>
  );
}

export function IconFlask(props: IconProps) {
  return (
    <Base {...props}>
      <path d="M8 3h4M9 3v5l-4.5 7.5A1.5 1.5 0 0 0 5.8 18h8.4a1.5 1.5 0 0 0 1.3-2.5L11 8V3" />
      <path d="M6.5 13h7" />
    </Base>
  );
}

export function IconGrid(props: IconProps) {
  return (
    <Base {...props}>
      <rect x="3" y="3" width="6" height="6" />
      <rect x="11" y="3" width="6" height="6" />
      <rect x="3" y="11" width="6" height="6" />
      <rect x="11" y="11" width="6" height="6" />
    </Base>
  );
}

export function IconPeople(props: IconProps) {
  return (
    <Base {...props}>
      <circle cx="7" cy="7" r="3" />
      <path d="M2.5 17c0-2.5 2-4.5 4.5-4.5s4.5 2 4.5 4.5" />
      <circle cx="14" cy="8" r="2.2" />
      <path d="M13 12.8c2.5 0 4.5 1.8 4.5 4.2" />
    </Base>
  );
}

export function IconDocs(props: IconProps) {
  return (
    <Base {...props}>
      <path d="M5 3h7l3 3v11H5z" />
      <path d="M12 3v3h3M8 10h5M8 13h5" />
    </Base>
  );
}

export function IconCheck(props: IconProps) {
  return (
    <Base {...props}>
      <path d="M4 10.5 8 14.5 16 5.5" />
    </Base>
  );
}

export function IconBook(props: IconProps) {
  return (
    <Base {...props}>
      <path d="M4 4c2-1 4-1 6 0v13c-2-1-4-1-6 0zM10 4c2-1 4-1 6 0v13c-2-1-4-1-6 0" />
    </Base>
  );
}

export function IconTheme(props: IconProps) {
  return (
    <Base {...props}>
      <circle cx="10" cy="10" r="6.5" />
      <path d="M10 3.5v13" />
      <path d="M10 3.5a6.5 6.5 0 0 1 0 13" fill="currentColor" stroke="none" />
    </Base>
  );
}

export function IconStamp(props: IconProps) {
  return (
    <Base {...props}>
      <rect x="3.5" y="3.5" width="13" height="13" />
      <rect x="6" y="6" width="8" height="8" strokeDasharray="2 2" />
    </Base>
  );
}

export function IconGraph(props: IconProps) {
  return (
    <Base {...props}>
      <circle cx="5" cy="15" r="2" />
      <circle cx="10" cy="5" r="2" />
      <circle cx="15" cy="12" r="2" />
      <path d="M6.5 13.5 8.8 6.8M11.8 6.2l2.2 4M6.9 14.6l6.2-1.8" />
    </Base>
  );
}

export function IconExport(props: IconProps) {
  return (
    <Base {...props}>
      <path d="M10 3v9M6.5 8.5 10 12l3.5-3.5" />
      <path d="M4 14v3h12v-3" />
    </Base>
  );
}
