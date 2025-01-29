import xw from "xwind";
import ButtonReact from "../components/ButtonReact";
import ButtonStyled from "../components/ButtonStyled";

const Index = () => (
  <div css={xw`grid justify-center items-center h-screen space-y-20`}>
    <div css={xw`space-y-20`}>
      <h1
        css={xw`mb-4 text-4xl font-extrabold leading-none tracking-tight text-gray-900 md:text-5xl lg:text-6xl dark:text-white`}
      >
        Chapter 4 - Tailwind CSS Example
      </h1>
      <p>
        The components below use the combination of Tailwind, xwind and Emotion
        to render the components. Emotion is a library designed for writing css
        styles with JavaScript.
      </p>
      <ButtonReact>This button is styled with React properties</ButtonReact>
      <ButtonStyled>
        This button is styled using the @emotion/styled package
      </ButtonStyled>
      <p>
        Edit the files in the <code>pages</code> and the <code>components</code>{" "}
        directory. Your changes will be automatically applied.
      </p>
    </div>
  </div>
);

export default Index;
